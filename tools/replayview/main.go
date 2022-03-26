package main

import (
	"errors"
	"flag"
	"image"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/oto/v2"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
)

var (
	romPath = flag.String("rom_path", "bn6.gba", "path to rom")
)

type Game struct {
	replayer *game.Replayer

	vb *av.VideoBuffer

	fbuf   *image.RGBA
	fbufMu sync.Mutex

	audioPlayer oto.Player

	t *mgba.Thread
}

func (g *Game) serviceFbuf() {
	runtime.LockOSThread()
	for {
		if g.replayer.Core.GBA().Sync().WaitFrameStart() {
			g.fbufMu.Lock()
			g.fbuf = g.vb.CopyImage()
			g.fbufMu.Unlock()
		} else {
			// TODO: Optimize this.
			time.Sleep(500 * time.Microsecond)
		}
		g.replayer.Core.GBA().Sync().WaitFrameEnd()
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.replayer.Core.DesiredVideoDimensions()
}

func (g *Game) Update() error {
	if g.t.HasCrashed() {
		return errors.New("mgba thread crashed")
	}
	g.audioPlayer.Play()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.fbufMu.Lock()
	defer g.fbufMu.Unlock()

	if g.fbuf == nil {
		return
	}

	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(ebiten.NewImageFromImage(g.fbuf), opts)
}

const expectedFPS = 60

func main() {
	flag.Parse()

	replayName := flag.Arg(0)
	f, err := os.Open(replayName)
	if err != nil {
		log.Fatalf("failed to open replay: %s", err)
	}

	replay, err := game.DeserializeReplay(f)
	if err != nil {
		log.Fatalf("failed to read replay: %s", err)
	}

	replayer, err := game.NewReplayer(*romPath, replay)
	if err != nil {
		log.Fatalf("failed to make replayer: %s", err)
	}

	audioCtx, ready, err := oto.NewContext(replayer.Core.Options().SampleRate, 2, 2)
	if err != nil {
		log.Fatalf("failed to initialize audio: %s", err)
	}
	<-ready

	width, height := replayer.Core.DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)
	ebiten.SetWindowSize(width*3, height*3)

	replayer.Core.SetVideoBuffer(vb.Pointer(), width)
	t := mgba.NewThread(replayer.Core)
	if !t.Start() {
		log.Fatalf("failed to start mgba thread")
	}
	replayer.Core.GBA().Sync().SetFPSTarget(float32(expectedFPS))

	audioPlayer := audioCtx.NewPlayer(av.NewAudioReader(replayer.Core, replayer.Core.Options().SampleRate))
	audioPlayer.(oto.BufferSizeSetter).SetBufferSize(replayer.Core.Options().AudioBuffers * 4)

	g := &Game{
		replayer:    replayer,
		vb:          vb,
		audioPlayer: audioPlayer,
		t:           t,
	}

	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowTitle("bbn6 replayview")
	ebiten.SetMaxTPS(ebiten.UncappedTPS)
	ebiten.SetWindowResizable(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	go g.serviceFbuf()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("failed to run mgba: %s", err)
	}
}
