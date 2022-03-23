package main

import (
	"errors"
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/oto/v2"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/iobuf"
	"github.com/murkland/bbn6/mgba"
)

var (
	romPath = flag.String("rom_path", "bn6f.gba", "path to rom")
)

type Game struct {
	core   *mgba.Core
	vb     *iobuf.VideoBuffer
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
		opts := &ebiten.DrawImageOptions{}
		screen.DrawImage(ebiten.NewImageFromImage(g.vb.CopyImage()), opts)
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

	vb := iobuf.NewVideoBuffer(width, height)
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

	core.InstallGBASWI16IRQHTraps(mgba.IRQTraps{0xff: bn6.MakeIRQFFTrap(core, offsets)})
	bn6.PatchWithSVCFFs(core, offsets)

	t := mgba.NewThread(core)
	if !t.Start() {
		log.Fatalf("failed to start mgba thread")
	}

	player := audioCtx.NewPlayer(iobuf.NewAudioReader(core, core.Options().SampleRate))

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
