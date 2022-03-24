package game

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/oto/v2"
	"github.com/keegancsmith/nth"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/packets"
	"github.com/murkland/bbn6/trapper"
	"github.com/murkland/ctxwebrtc"
	"github.com/murkland/ringbuf"
	"golang.org/x/exp/constraints"
	"golang.org/x/sync/errgroup"
)

type Game struct {
	conf config.Config

	dc *ctxwebrtc.DataChannel

	mainCore      *mgba.Core
	fastforwarder *fastforwarder

	vb   *av.VideoBuffer
	fbuf *ebiten.Image

	audioPlayer oto.Player

	t *mgba.Thread

	isAnswerer bool // TODO: negotiate this

	mustCommitNewState bool
	pendingState       *mgba.State
	pendingRemoteInit  []byte
	battle             *Battle

	delayRingbuf   *ringbuf.RingBuf[time.Duration]
	delayRingbufMu sync.RWMutex
}

func New(conf config.Config, romPath string, dc *ctxwebrtc.DataChannel, isAnswerer bool) (*Game, error) {
	mainCore, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	mainCore.AutoloadSave()

	offsets, ok := bn6.OffsetsForGame(mainCore.GameTitle())
	if !ok {
		return nil, fmt.Errorf("unsupported game: %s", mainCore.GameTitle())
	}

	fastforwarder, err := newFastforwarder(romPath, offsets)
	if err != nil {
		return nil, err
	}

	audioCtx, ready, err := oto.NewContext(mainCore.Options().SampleRate, 2, 2)
	if err != nil {
		return nil, err
	}
	<-ready
	audioCtx.SetReadBufferSize(mainCore.Options().AudioBuffers * 4)

	width, height := mainCore.DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)

	mainCore.SetVideoBuffer(vb.Pointer(), width)
	t := mgba.NewThread(mainCore)
	if !t.Start() {
		log.Fatalf("failed to start mgba thread")
	}

	audioPlayer := audioCtx.NewPlayer(av.NewAudioReader(mainCore, mainCore.Options().SampleRate))

	g := &Game{
		conf: conf,
		dc:   dc,

		mainCore:      mainCore,
		fastforwarder: fastforwarder,

		vb:   vb,
		fbuf: ebiten.NewImage(width, height),

		audioPlayer: audioPlayer,

		t: t,

		isAnswerer: isAnswerer,

		delayRingbuf: ringbuf.New[time.Duration](10),
	}
	g.InstallTraps(mainCore)

	return g, nil
}

func (g *Game) RunBackgroundTasks(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() error {
		return g.handleConn(ctx)
	})

	errg.Go(func() error {
		return g.sendPings(ctx)
	})

	return errg.Wait()
}

func (g *Game) sendPings(ctx context.Context) error {
	for {
		now := time.Now()
		if err := packets.Send(ctx, g.dc, packets.Ping{
			ID: uint64(now.UnixMicro()),
		}, nil); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

type orderableSlice[T constraints.Ordered] []T

func (s orderableSlice[T]) Len() int {
	return len(s)
}

func (s orderableSlice[T]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s orderableSlice[T]) Less(i, j int) bool {
	return s[i] < s[j]
}

func (g *Game) medianDelay() time.Duration {
	g.delayRingbufMu.RLock()
	defer g.delayRingbufMu.RUnlock()

	if g.delayRingbuf.Used() == 0 {
		return 0
	}

	delays := make([]time.Duration, g.delayRingbuf.Used())
	g.delayRingbuf.Peek(delays, 0)

	i := len(delays) / 2
	nth.Element(orderableSlice[time.Duration](delays), i)
	return delays[i]
}

func (g *Game) handleConn(ctx context.Context) error {
	for {
		packet, trailer, err := packets.Recv(ctx, g.dc)
		if err != nil {
			return err
		}

		switch p := packet.(type) {
		case packets.Ping:
			if err := packets.Send(ctx, g.dc, packets.Pong{ID: p.ID}, nil); err != nil {
				return err
			}
		case packets.Pong:
			if err := (func() error {
				g.delayRingbufMu.Lock()
				defer g.delayRingbufMu.Unlock()

				if g.delayRingbuf.Free() == 0 {
					g.delayRingbuf.Advance(1)
				}

				delay := time.Now().Sub(time.UnixMicro(int64(p.ID)))
				g.delayRingbuf.Push([]time.Duration{delay})
				return nil
			})(); err != nil {
				return err
			}
		case packets.Init:
			g.pendingRemoteInit = p.Marshaled[:]
		case packets.Input:
			if err := (func() error {
				g.battle.mu.Lock()
				defer g.battle.mu.Unlock()

				g.battle.iq.AddInput(g.battle.RemotePlayerIndex(), Input{int(p.ForTick), p.Joyflags, p.CustomScreenState, trailer})
				return nil
			})(); err != nil {
				return err
			}
		}
	}
}

func (g *Game) InstallTraps(core *mgba.Core) error {
	offsets, ok := bn6.OffsetsForGame(core.GameTitle())
	if !ok {
		return fmt.Errorf("unsupported game: %s", core.GameTitle())
	}

	tp := trapper.New(core)

	tp.Add(offsets.A_battle_init__call__battle_copyInputData, func() {
		if g.battle == nil {
			return
		}

		g.battle.mu.Lock()
		defer g.battle.mu.Unlock()

		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		if g.pendingRemoteInit == nil {
			return
		}

		bn6.SetPlayerMarshaledBattleState(core, g.battle.RemotePlayerIndex(), g.pendingRemoteInit)
	})

	tp.Add(offsets.A_battle_init_marshal__ret, func() {
		if g.battle == nil {
			return
		}

		g.battle.mu.Lock()
		defer g.battle.mu.Unlock()

		ctx := context.Background()

		marshaled := bn6.LocalMarshaledBattleState(core)

		var pkt packets.Init
		copy(pkt.Marshaled[:], marshaled)
		if err := packets.Send(ctx, g.dc, pkt, nil); err != nil {
			panic(err)
		}
		bn6.SetPlayerMarshaledBattleState(core, g.battle.LocalPlayerIndex(), marshaled)
	})

	tp.Add(offsets.A_battle_turn_marshal__ret, func() {
		if g.battle == nil {
			return
		}

		g.battle.mu.Lock()
		defer g.battle.mu.Unlock()

		g.battle.localPendingTurn = bn6.LocalMarshaledBattleState(core)
	})

	tp.Add(offsets.A_battle_update__call__battle_copyInputData, func() {
		if g.battle == nil {
			return
		}

		g.battle.mu.Lock()
		defer g.battle.mu.Unlock()

		ctx := context.Background()

		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		if g.battle.startFrame == 0 {
			g.battle.startFrame = core.FrameCounter()
			g.mustCommitNewState = true
			return
		}

		tick := core.FrameCounter() - g.battle.startFrame

		joyflags := bn6.LocalJoyflags(core)
		customScreenState := bn6.LocalCustomScreenState(core)
		turn := g.battle.localPendingTurn
		g.battle.localPendingTurn = nil

		var pkt packets.Input
		pkt.ForTick = tick
		pkt.Joyflags = joyflags
		pkt.CustomScreenState = customScreenState
		if err := packets.Send(ctx, g.dc, pkt, turn); err != nil {
			panic(err)
		}

		g.battle.iq.AddInput(g.battle.LocalPlayerIndex(), Input{int(tick), joyflags, customScreenState, turn})
		inputPairs := g.battle.iq.Consume()
		if len(inputPairs) > 0 {
			left := g.battle.iq.Peek(g.battle.LocalPlayerIndex())

			committedState, dirtyState, err := g.fastforwarder.fastforward(g.battle.committedState, g.battle.LocalPlayerIndex(), inputPairs, left)
			if err != nil {
				panic(err)
			}
			g.battle.committedState = committedState
			g.pendingState = dirtyState
		}

		bn6.SetPlayerInputState(core, g.battle.LocalPlayerIndex(), joyflags, customScreenState)
		if turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, g.battle.LocalPlayerIndex(), turn)
		}
	})

	tp.Add(offsets.A_battle_updating__ret__go_to_custom_screen, func() {
		if g.battle == nil {
			return
		}

		g.battle.mu.Lock()
		defer g.battle.mu.Unlock()

		tick := core.FrameCounter() - g.battle.startFrame
		log.Printf("turn ended on %df, rng state = %08x", tick, bn6.RNG2State(core))
	})

	tp.Add(offsets.A_battle_start__ret, func() {
		log.Printf("battle started")
		g.battle = NewBattle(g.isAnswerer)
	})

	tp.Add(offsets.A_battle_end__entry, func() {
		log.Printf("battle ended")
		g.battle = nil
	})

	tp.Add(offsets.A_battle_isP2__tst, func() {
		if g.battle == nil {
			return
		}

		core.GBA().SetRegister(0, uint32(g.battle.LocalPlayerIndex()))
	})

	tp.Add(offsets.A_link_isP2__ret, func() {
		if g.battle == nil {
			return
		}

		core.GBA().SetRegister(0, uint32(g.battle.LocalPlayerIndex()))
	})

	tp.Add(offsets.A_commMenu_handleLinkCableInput__entry, func() {
		log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", core.GBA().Register(15)-4)
	})

	tp.Add(offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, func() {
		bn6.StartBattleFromCommMenu(core)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(offsets.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})
	core.InstallBeefTrap(tp.BeefHandler)

	return nil
}

func (g *Game) Finish() {
	g.t.End()
	g.t.Join()
}

func (g *Game) Update() error {
	if g.t.HasCrashed() {
		return errors.New("mgba thread crashed")
	}

	g.audioPlayer.Play()

	if g.battle != nil {
		g.battle.mu.Lock()
		defer g.battle.mu.Unlock()

		highWaterMark := int(g.medianDelay()*time.Duration(60)/2/time.Second + 1)
		if highWaterMark < 1 {
			highWaterMark = 1
		}

		if g.battle.iq.Lag(g.battle.RemotePlayerIndex()) >= highWaterMark {
			// Pause until we have enough space.
			return nil
		}
	}

	var keys mgba.Keys
	for _, key := range inpututil.AppendPressedKeys(nil) {
		if key == g.conf.Keymapping.A {
			keys |= mgba.KeysA
		}
		if key == g.conf.Keymapping.B {
			keys |= mgba.KeysB
		}
		if key == g.conf.Keymapping.L {
			keys |= mgba.KeysL
		}
		if key == g.conf.Keymapping.R {
			keys |= mgba.KeysR
		}
		if key == g.conf.Keymapping.Left {
			keys |= mgba.KeysLeft
		}
		if key == g.conf.Keymapping.Right {
			keys |= mgba.KeysRight
		}
		if key == g.conf.Keymapping.Up {
			keys |= mgba.KeysUp
		}
		if key == g.conf.Keymapping.Down {
			keys |= mgba.KeysDown
		}
		if key == g.conf.Keymapping.Start {
			keys |= mgba.KeysStart
		}
		if key == g.conf.Keymapping.Select {
			keys |= mgba.KeysSelect
		}
	}
	g.mainCore.SetKeys(keys)

	if g.mainCore.GBA().Sync().WaitFrameStart() {
		g.fbuf.Fill(color.White)
		img := g.vb.CopyImage()
		for i := range img.Pix {
			if i%4 == 3 {
				img.Pix[i] = 0xff
			}
		}
		opts := &ebiten.DrawImageOptions{}
		g.fbuf.DrawImage(ebiten.NewImageFromImage(img), opts)

		if g.mustCommitNewState {
			g.battle.committedState = g.mainCore.SaveState()
		}
		g.mustCommitNewState = false

		if g.pendingState != nil {
			g.mainCore.LoadState(g.pendingState)
		}
		g.pendingState = nil
	}
	g.mainCore.GBA().Sync().WaitFrameEnd()

	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.mainCore.DesiredVideoDimensions()
}

func (g *Game) Draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(g.fbuf, opts)
}
