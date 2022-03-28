package game

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/packets"
	"github.com/murkland/bbn6/trapper"
	"github.com/ncruces/zenity"
	"golang.org/x/exp/constraints"
	"golang.org/x/sync/errgroup"
)

type Game struct {
	conf config.Config

	mainCore      *mgba.Core
	fastforwarder *fastforwarder

	bn6 *bn6.BN6

	vb *av.VideoBuffer

	fbuf   *image.RGBA
	fbufMu sync.Mutex

	audioCtx        *audio.Context
	gameAudioPlayer *audio.Player

	t *mgba.Thread

	match   *Match
	matchMu sync.Mutex

	debugSpew bool
}

func New(conf config.Config, romPath string) (*Game, error) {
	mainCore, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	mainCore.AutoloadSave()

	bn6 := bn6.Load(mainCore.GameTitle())
	if bn6 == nil {
		return nil, fmt.Errorf("unsupported game: %s", mainCore.GameTitle())
	}

	fastforwarder, err := newFastforwarder(romPath, bn6)
	if err != nil {
		return nil, err
	}

	audioCtx := audio.NewContext(mainCore.Options().SampleRate)

	width, height := mainCore.DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)
	ebiten.SetWindowSize(width*3, height*3)

	mainCore.SetVideoBuffer(vb.Pointer(), width)
	t := mgba.NewThread(mainCore)
	if !t.Start() {
		log.Fatalf("failed to start mgba thread")
	}
	mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))

	gameAudioPlayer, err := audioCtx.NewPlayer(av.NewAudioReader(mainCore, mainCore.Options().SampleRate))
	if err != nil {
		return nil, err
	}
	gameAudioPlayer.SetBufferSize(time.Duration(mainCore.Options().AudioBuffers+0x4) * time.Second / time.Duration(mainCore.Options().SampleRate))
	gameAudioPlayer.Play()

	g := &Game{
		conf: conf,

		mainCore:      mainCore,
		fastforwarder: fastforwarder,

		bn6: bn6,

		vb: vb,

		audioCtx:        audioCtx,
		gameAudioPlayer: gameAudioPlayer,

		t: t,
	}
	g.InstallTraps(mainCore)

	return g, nil
}

func (g *Game) RunBackgroundTasks(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() error {
		g.serviceFbuf()
		return nil
	})

	return errg.Wait()
}

func (g *Game) serviceFbuf() {
	runtime.LockOSThread()
	for {
		if g.mainCore.GBA().Sync().WaitFrameStart() {
			g.fbufMu.Lock()
			g.fbuf = g.vb.CopyImage()
			g.fbufMu.Unlock()
		} else {
			// TODO: Optimize this.
			time.Sleep(500 * time.Microsecond)
		}
		g.mainCore.GBA().Sync().WaitFrameEnd()
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

func (g *Game) InstallTraps(core *mgba.Core) error {
	tp := trapper.New(core)

	tp.Add(g.bn6.Offsets.ROM.A_battle_init__call__battle_copyInputData, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		core.GBA().SetRegister(0, 0x0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()

		if g.match.battle.remoteInit == nil {
			return
		}

		g.bn6.SetPlayerMarshaledBattleState(core, g.match.battle.RemotePlayerIndex(), g.match.battle.remoteInit)
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_init_marshal__ret, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		ctx := context.Background()

		g.match.battle.localInit = g.bn6.LocalMarshaledBattleState(core)

		var pkt packets.Init
		copy(pkt.Marshaled[:], g.match.battle.localInit)
		if err := packets.Send(ctx, g.match.dc, pkt, nil); err != nil {
			panic(err)
		}
		log.Printf("init sent")

		g.bn6.SetPlayerMarshaledBattleState(core, g.match.battle.LocalPlayerIndex(), g.match.battle.localInit)
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_turn_marshal__ret, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		g.match.battle.localPendingTurn = g.bn6.LocalMarshaledBattleState(core)
		g.match.battle.localPendingTurnWaitTicksLeft = 64
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		core.GBA().SetRegister(0, 0x0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()

		ctx := context.Background()
		if g.match.battle.committedState == nil {
			g.match.battle.committedState = core.SaveState()
			if err := g.match.battle.rw.WriteState(g.match.battle.LocalPlayerIndex(), g.match.battle.committedState); err != nil {
				panic(err)
			}
			if err := g.match.battle.rw.WriteInit(g.match.battle.LocalPlayerIndex(), g.match.battle.localInit); err != nil {
				panic(err)
			}
			if err := g.match.battle.rw.WriteInit(g.match.battle.RemotePlayerIndex(), g.match.battle.remoteInit); err != nil {
				panic(err)
			}
			return
		}

		g.match.battle.tick++

		nextJoyflags := g.bn6.LocalJoyflags(core)
		g.match.battle.localInputBuffer.Push([]uint16{nextJoyflags})

		joyflags := uint16(0xfc00)
		if g.match.battle.localInputBuffer.Free() == 0 {
			var joyflagsBuf [1]uint16
			g.match.battle.localInputBuffer.Pop(joyflagsBuf[:], 0)
			joyflags = joyflagsBuf[0]
		}

		customScreenState := g.bn6.LocalCustomScreenState(core)

		var turn []byte
		if g.match.battle.localPendingTurnWaitTicksLeft > 0 {
			g.match.battle.localPendingTurnWaitTicksLeft--
			if g.match.battle.localPendingTurnWaitTicksLeft == 0 {
				turn = g.match.battle.localPendingTurn
				g.match.battle.localPendingTurn = nil
			}
		}

		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := g.match.battle.iq.AddInput(ctx, g.match.battle.LocalPlayerIndex(), Input{int(g.match.battle.tick), joyflags, customScreenState, turn}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				log.Printf("could not queue local input within %s, dropping connection", timeout)
				g.match.Close()
				g.match = nil
				g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
				// TODO: Figure out how to gracefully exit the battle.
				panic(err)
			}
			panic(err)
		}

		var pkt packets.Input
		pkt.ForTick = uint32(g.match.battle.tick)
		pkt.Joyflags = joyflags
		pkt.CustomScreenState = customScreenState
		if err := packets.Send(ctx, g.match.dc, pkt, turn); err != nil {
			panic(err)
		}

		inputPairs := g.match.battle.iq.Consume()
		if len(inputPairs) > 0 {
			g.match.battle.lastCommittedRemoteInput = inputPairs[len(inputPairs)-1][1-g.match.battle.LocalPlayerIndex()]
		}

		left := g.match.battle.iq.Peek(g.match.battle.LocalPlayerIndex())
		committedState, dirtyState, err := g.fastforwarder.fastforward(g.match.battle.committedState, g.match.battle.rw, g.match.battle.LocalPlayerIndex(), inputPairs, g.match.battle.lastCommittedRemoteInput, left)
		if err != nil {
			panic(err)
		}
		g.match.battle.committedState = committedState
		g.mainCore.LoadState(dirtyState)
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_runUnpausedStep__cmp__retval, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		r := core.GBA().Register(0)
		if r == 1 {
			g.match.wonLastBattle = true
		} else if r == 2 {
			g.match.wonLastBattle = false
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_updating__ret__go_to_custom_screen, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		tick := g.match.battle.tick
		log.Printf("turn ended on %d, rng state = %08x", tick, g.bn6.RNG2State(core))
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_start__ret, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		if g.match.battle != nil {
			panic("battle already started?")
		}

		g.match.battleNumber++
		log.Printf("battle %d started, won last battle (is p1) = %t", g.match.battleNumber, g.match.wonLastBattle)

		const localInputBufferSize = 2
		battle, err := NewBattle(g.mainCore, !g.match.wonLastBattle, localInputBufferSize)
		if err != nil {
			panic(err)
		}
		g.match.battle = battle
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_end__entry, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		log.Printf("battle ended, won = %t", g.match.wonLastBattle)
		if err := g.match.battle.Close(); err != nil {
			panic(err)
		}
		g.match.battle = nil

		g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		core.GBA().SetRegister(0, uint32(g.match.battle.LocalPlayerIndex()))
	})

	tp.Add(g.bn6.Offsets.ROM.A_link_isP2__ret, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			return
		}

		core.GBA().SetRegister(0, uint32(g.match.battle.LocalPlayerIndex()))
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_handleLinkCableInput__entry, func() {
		log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", core.GBA().Register(15)-4)
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, func() {
		ctx := context.Background()

		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			volume := g.gameAudioPlayer.Volume()
			g.gameAudioPlayer.SetVolume(0)
			code, err := zenity.Entry("Enter a code to matchmake with:", zenity.Title("bbn6"))
			g.gameAudioPlayer.SetVolume(volume)
			if err != nil {
				log.Printf("matchmaking dialog did not return a code: %s", err)
				g.bn6.DropMatchmakingFromCommMenu(core)
			} else {
				match, err := NewMatch(g.conf, code, 0)
				if err != nil {
					// TODO: handle this better.
					panic(err)
				}
				g.match = match
				go g.match.Run(ctx)
			}
		}

		if g.match != nil {
			select {
			case <-g.match.connReady:
				g.bn6.StartBattleFromCommMenu(core)
				log.Printf("match started")
			default:
			}
		}

		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_waitForFriend__ret__cancel, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		g.match.Close()
		g.match = nil
		log.Printf("match canceled by user")

		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_endBattle__entry, func() {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		log.Printf("match ended")
		g.match.Close()
		g.match = nil
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	core.InstallBeefTrap(tp.BeefHandler)

	return nil
}

func (g *Game) Finish() {
	g.t.End()
	g.t.Join()
}

const expectedFPS = 60

func (g *Game) runaheadTicksAllowedMatchLocked() int {
	expected := int(g.match.medianDelay()*time.Duration(expectedFPS)/2/time.Second + 1)
	if expected < 1 {
		expected = 1
	}
	return expected
}

func (g *Game) Update() error {
	if g.t.HasCrashed() {
		return errors.New("mgba thread crashed")
	}

	if err := (func() error {
		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match != nil && g.match.battle != nil {
			expected := g.runaheadTicksAllowedMatchLocked()
			lag := g.match.battle.iq.Lag(g.match.battle.RemotePlayerIndex())
			tps := expectedFPS - (lag - expected + 1)
			// TODO: Not thread safe.
			g.mainCore.GBA().Sync().SetFPSTarget(float32(tps))
		}

		return nil
	})(); err != nil {
		return err
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

	if g.conf.Keymapping.DebugSpew != -1 && inpututil.IsKeyJustPressed(g.conf.Keymapping.DebugSpew) {
		g.debugSpew = !g.debugSpew
	}

	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.mainCore.DesiredVideoDimensions()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.fbufMu.Lock()
	defer g.fbufMu.Unlock()

	if g.fbuf == nil {
		return
	}

	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(ebiten.NewImageFromImage(g.fbuf), opts)

	if g.debugSpew {
		g.spewDebug(screen)
	}
}
