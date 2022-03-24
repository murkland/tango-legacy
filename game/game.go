package game

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/oto/v2"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/trapper"
	"github.com/murkland/ctxwebrtc"
)

type Game struct {
	dc *ctxwebrtc.DataChannel

	mainCore     *mgba.Core
	steppingCore *mgba.Core

	vb          *av.VideoBuffer
	t           *mgba.Thread
	audioPlayer oto.Player

	isAnswerer bool // TODO: negotiate this

	battle *Battle
}

var coreOptions = mgba.CoreOptions{
	SampleRate:   48000,
	AudioBuffers: 1024,
	AudioSync:    true,
	VideoSync:    true,
	Volume:       0x100,
}

func newCore(romPath string) (*mgba.Core, error) {
	core, err := mgba.FindCore(romPath)
	if err != nil {
		return nil, err
	}
	core.SetOptions(coreOptions)

	if err := core.LoadFile(romPath); err != nil {
		return nil, err
	}

	core.Config().Init("bbn6")
	core.Config().Load()
	core.LoadConfig()

	return core, nil
}

func New(romPath string, dc *ctxwebrtc.DataChannel, isAnswerer bool) (*Game, error) {
	mainCore, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	steppingCore, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	mainCore.AutoloadSave()

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

	g := &Game{dc, mainCore, steppingCore, vb, t, audioPlayer, isAnswerer, nil}
	g.InstallTraps(mainCore)

	return g, nil
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

		ctx := context.Background()

		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		init, err := g.battle.ReceiveInit(ctx, g.dc)
		if err != nil {
			panic(err)
		}

		if init == nil {
			return
		}

		bn6.SetPlayerMarshaledBattleState(core, g.battle.RemotePlayerIndex(), init)
	})

	tp.Add(offsets.A_battle_update__call__battle_copyInputData, func() {
		if g.battle == nil {
			return
		}

		ctx := context.Background()

		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		if g.battle.StartFrameNumber == 0 {
			g.battle.StartFrameNumber = core.FrameCounter()
		}

		tick := int(core.FrameCounter() - g.battle.StartFrameNumber)
		g.battle.LastTick = tick

		joyflags := bn6.LocalJoyflags(core)
		customScreenState := bn6.LocalCustomScreenState(core)

		if err := g.battle.QueueLocalInput(ctx, g.dc, tick, joyflags, customScreenState); err != nil {
			panic(err)
		}

		local, remote, err := g.battle.DequeueInputs(ctx, g.dc)
		if err != nil {
			panic(err)
		}

		if local.Tick != remote.Tick {
			panic(fmt.Sprintf("local tick != remote tick: %d != %d", local.Tick, remote.Tick))
		}

		bn6.SetPlayerInputState(core, g.battle.LocalPlayerIndex(), local.Joyflags, local.CustomScreenState)
		bn6.SetPlayerInputState(core, g.battle.RemotePlayerIndex(), remote.Joyflags, remote.CustomScreenState)

		if local.Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, g.battle.LocalPlayerIndex(), local.Turn)
			log.Printf("local turn committed on %df", tick)
		}

		if remote.Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, g.battle.RemotePlayerIndex(), remote.Turn)
			log.Printf("remote turn committed on %df", tick)
		}
	})

	tp.Add(offsets.A_battle_init_marshal__ret, func() {
		if g.battle == nil {
			return
		}

		ctx := context.Background()

		init := bn6.LocalMarshaledBattleState(core)
		if err := g.battle.SendInit(ctx, g.dc, init); err != nil {
			panic(err)
		}
		bn6.SetPlayerMarshaledBattleState(core, g.battle.LocalPlayerIndex(), init)
	})

	tp.Add(offsets.A_battle_turn_marshal__ret, func() {
		if g.battle == nil {
			return
		}

		ctx := context.Background()

		tick := int(core.FrameCounter() - g.battle.StartFrameNumber)
		turn := bn6.LocalMarshaledBattleState(core)
		log.Printf("sending turn data on %df", tick)
		if err := g.battle.QueueLocalTurn(ctx, g.dc, tick, turn); err != nil {
			panic(err)
		}
	})

	tp.Add(offsets.A_battle_updating__ret__go_to_custom_screen, func() {
		if g.battle == nil {
			return
		}

		tick := int(core.FrameCounter() - g.battle.StartFrameNumber)
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

	var keys mgba.Keys
	for _, key := range inpututil.AppendPressedKeys(nil) {
		switch key {
		case ebiten.KeyZ:
			keys |= mgba.KeysA
		case ebiten.KeyX:
			keys |= mgba.KeysB
		case ebiten.KeyA:
			keys |= mgba.KeysL
		case ebiten.KeyS:
			keys |= mgba.KeysR
		case ebiten.KeyLeft:
			keys |= mgba.KeysLeft
		case ebiten.KeyRight:
			keys |= mgba.KeysRight
		case ebiten.KeyUp:
			keys |= mgba.KeysUp
		case ebiten.KeyDown:
			keys |= mgba.KeysDown
		case ebiten.KeyEnter:
			keys |= mgba.KeysStart
		case ebiten.KeyBackspace:
			keys |= mgba.KeysSelect
		}
	}
	g.mainCore.SetKeys(keys)

	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.mainCore.DesiredVideoDimensions()
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.mainCore.GBA().Sync().WaitFrameStart() {
		screen.Fill(color.White)
		opts := &ebiten.DrawImageOptions{}
		img := g.vb.Image()
		for i := range img.Pix {
			if i%4 == 3 {
				img.Pix[i] = 0xff
			}
		}
		screen.DrawImage(ebiten.NewImageFromImage(img), opts)
	}
	g.mainCore.GBA().Sync().WaitFrameEnd()
}
