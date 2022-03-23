package main

import (
	"errors"
	"flag"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/oto/v2"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/trapper"
)

var (
	romPath = flag.String("rom_path", "bn6f.gba", "path to rom")
)

type Game struct {
	core   *mgba.Core
	vb     *av.VideoBuffer
	t      *mgba.Thread
	player oto.Player
}

func (g *Game) Update() error {
	if g.t.HasCrashed() {
		return errors.New("mgba thread crashed")
	}

	g.player.Play()

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
	g.core.SetKeys(keys)

	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.core.DesiredVideoDimensions()
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.core.GBA().Sync().WaitFrameStart() {
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
	g.core.GBA().Sync().WaitFrameEnd()
}

func main() {
	flag.Parse()

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x3 == 0 {
			return
		}
		log.Printf("level=%d category=%s %s", level, category, message)
	})

	core, err := mgba.FindCore(*romPath)
	if err != nil {
		log.Fatalf("failed to start mgba: %s", err)
	}

	core.SetOptions(mgba.CoreOptions{
		SampleRate:   48000,
		AudioBuffers: 1024,
		AudioSync:    true,
		VideoSync:    true,
		Volume:       0x100,
	})

	audioCtx, ready, err := oto.NewContext(core.Options().SampleRate, 2, 2)
	if err != nil {
		log.Fatalf("failed to acquire audio context: %s", err)
	}
	<-ready
	audioCtx.SetReadBufferSize(core.Options().AudioBuffers * 4)

	width, height := core.DesiredVideoDimensions()
	log.Printf("width = %d, height = %d", width, height)

	vb := av.NewVideoBuffer(width, height)
	core.SetVideoBuffer(vb.Pointer(), width)

	if err := core.LoadFile(*romPath); err != nil {
		log.Fatalf("failed to start mgba: %s", err)
	}

	log.Printf("game code: %s, game title: %s", core.GameCode(), core.GameTitle())
	offsets, ok := bn6.OffsetsForGame(core.GameTitle())
	if !ok {
		log.Fatalf("unsupported game")
	}

	core.Config().Init("bbn6")
	core.Config().Load()
	core.LoadConfig()
	if core.AutoloadSave() {
		log.Printf("save autoload successful!")
	} else {
		log.Printf("failed to autoload save: is there a save file present?")
	}

	tp := trapper.New(core)

	tp.Add(offsets.A_battle_init__call__battle_copyInputData, func() {
		// TODO: Set this correctly.
		inLinkBattle := false

		if !inLinkBattle {
			return
		}

		// TODO: Get inputs.
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(offsets.A_battle_update__call__battle_copyInputData, func() {
		// TODO: Set this correctly.
		inLinkBattle := false

		if !inLinkBattle {
			return
		}

		// TODO: Get inputs.
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(offsets.A_battle_init_marshal__ret, func() {
		// TODO
		init := bn6.LocalMarshaledBattleState(core)
		log.Printf("battle init: %v", init)
	})

	tp.Add(offsets.A_battle_turn_marshal__ret, func() {
		// TODO
		turn := bn6.LocalMarshaledBattleState(core)
		log.Printf("battle turn: %v", turn)
	})

	tp.Add(offsets.A_battle_updating__ret__go_to_custom_screen, func() {
		// TODO
	})

	tp.Add(offsets.A_battle_start__ret, func() {
		// TODO
	})

	tp.Add(offsets.A_battle_end__entry, func() {
		// TODO
	})

	tp.Add(offsets.A_battle_isRemote__tst, func() {
		// TODO: Set isRemote
		isRemote := true
		_ = isRemote
	})

	tp.Add(offsets.A_link_isRemote__ret, func() {
		// TODO: Set isRemote
		isRemote := true
		if isRemote {
			core.GBA().SetRegister(0, 1)
		} else {
			core.GBA().SetRegister(0, 0)
		}
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

	t := mgba.NewThread(core)
	if !t.Start() {
		log.Fatalf("failed to start mgba thread")
	}

	player := audioCtx.NewPlayer(av.NewAudioReader(core, core.Options().SampleRate))

	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowTitle("bbn6")
	ebiten.SetMaxTPS(ebiten.UncappedTPS)
	ebiten.SetWindowResizable(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	if err := ebiten.RunGame(&Game{core, vb, t, player}); err != nil {
		log.Fatalf("failed to start mgba: %s", err)
	}

	t.End()
	t.Join()
}
