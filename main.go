package main

import (
	"C"
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/murkland/bbn6/mgba"
)
import "github.com/murkland/bbn6/bn6"

var (
	romPath = flag.String("rom_path", "bn6f.gba", "path to rom")
)

type Game struct {
	core *mgba.Core
	vb   *VideoBuffer
}

func (g *Game) Update() error {
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
	g.core.RunFrame()
	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.core.DesiredVideoDimensions()
}

func (g *Game) Draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	// Perform alpha correction: sometimes mGBA gives us white pixels when it shouldn't.
	opts.ColorM.Scale(1.0, 1.0, 1.0, 0.0)
	opts.ColorM.Translate(0.0, 0.0, 0.0, 1.0)
	screen.DrawImage(ebiten.NewImageFromImage(g.vb.RGBAImage()), opts)
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

	core.InstallGBASWI16IRQHTrap(func(imm int) bool {
		if imm == 0xff {
			log.Fatalf("serviced custom IRQ trap: %d at %08x", imm, core.Register(15))
		}
		return true
	})

	width, height := core.DesiredVideoDimensions()
	log.Printf("width = %d, height = %d", width, height)

	vb := NewVideoBuffer(width, height)
	core.SetVideoBuffer(vb.Pointer(), width)

	if err := core.LoadFile(*romPath); err != nil {
		log.Fatalf("failed to start mgba: %s", err)
	}

	log.Printf("game code: %s, game title: %s", core.GameCode(), core.GameTitle())
	offsets, ok := bn6.OffsetsForGame(core.GameTitle())
	if !ok {
		log.Fatalf("unsupported game")
	}

	core.RawWrite8(offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, -1, 0xff)
	core.RawWrite8(offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput+1, -1, 0xdf)

	core.ConfigInit("bbn6")
	core.ConfigLoad()
	core.LoadConfig()
	if core.AutoloadSave() {
		log.Printf("save autoload successful!")
	} else {
		log.Printf("failed to autoload save: is there a save file present?")
	}

	core.Reset()

	ebiten.SetWindowTitle("bbn6")
	ebiten.SetMaxTPS(ebiten.UncappedTPS)
	ebiten.SetWindowResizable(true)
	if err := ebiten.RunGame(&Game{core, vb}); err != nil {
		log.Fatalf("failed to start mgba: %s", err)
	}
}
