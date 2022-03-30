package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
	"github.com/ncruces/zenity"
)

var (
	romPath = flag.String("rom_path", "bn6.gba", "path to rom")
)

type Game struct {
	replayer *game.Replayer

	vb *av.VideoBuffer

	fbuf   *image.RGBA
	fbufMu sync.Mutex

	gameAudioPlayer *audio.Player

	t *mgba.Thread
}

func (g *Game) serviceFbuf() {
	runtime.LockOSThread()
	for {
		g.replayer.Core().SetKeys(mgba.Keys(g.replayer.PeekLocalJoyflags() & ^uint16(0xfc00)))
		if g.replayer.Core().GBA().Sync().WaitFrameStart() {
			g.fbufMu.Lock()
			g.fbuf = g.vb.CopyImage()
			g.fbufMu.Unlock()
		} else {
			// TODO: Optimize this.
			time.Sleep(500 * time.Microsecond)
		}
		g.replayer.Core().GBA().Sync().WaitFrameEnd()
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.replayer.Core().DesiredVideoDimensions()
}

func (g *Game) Update() error {
	if g.t.HasCrashed() {
		return errors.New("mgba thread crashed")
	}
	g.gameAudioPlayer.Play()

	fpsTarget := g.replayer.Core().GBA().Sync().FPSTarget()
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.replayer.Core().GBA().Sync().SetFPSTarget(fpsTarget + 10)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.replayer.Core().GBA().Sync().SetFPSTarget(fpsTarget - 10)
	}

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

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	replayName := flag.Arg(0)
	if replayName == "" {
		fn, err := zenity.SelectFile(zenity.Title("Select a replay to watch"))
		if err != nil {
			log.Fatalf("failed to prompt for replay: %s", err)
		}
		replayName = fn
	}

	f, err := os.Open(replayName)
	if err != nil {
		log.Fatalf("failed to open replay: %s", err)
	}
	defer f.Close()

	replay, err := game.DeserializeReplay(f)
	if err != nil {
		log.Fatalf("failed to open replay: %s", err)
	}

	roms, err := os.ReadDir("roms")
	if err != nil {
		log.Fatalf("failed to open roms directory: %s", err)
	}

	var romPath string
	for _, dirent := range roms {
		path := filepath.Join("roms", dirent.Name())

		if err := func() error {
			core, err := mgba.FindCore(path)
			if err != nil {
				return err
			}
			core.Config().Init("bbn6")
			defer core.Close()

			if err := core.LoadFile(path); err != nil {
				return err
			}

			if replay.State.ROMTitle != core.GameTitle() {
				return fmt.Errorf("rom title doesn't match: %s != %s", replay.State.ROMTitle, core.GameTitle())
			}

			if replay.State.ROMCRC32 != core.CRC32() {
				return fmt.Errorf("crc32 doesn't match: %08x != %08x", replay.State.ROMCRC32, core.GBA().ROMCRC32())
			}

			return nil
		}(); err != nil {
			log.Printf("%s not eligible: %s", path, err)
			continue
		}

		romPath = path
		break
	}

	if romPath == "" {
		log.Fatalf("failed find eligible rom")
	}

	replayer, err := game.NewReplayer(romPath, replay)
	if err != nil {
		log.Fatalf("failed to make replayer: %s", err)
	}

	audioCtx := audio.NewContext(replayer.Core().Options().SampleRate)

	width, height := replayer.Core().DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)
	ebiten.SetWindowSize(width*3, height*3)

	replayer.Core().SetVideoBuffer(vb.Pointer(), width)
	t := mgba.NewThread(replayer.Core())
	if !t.Start() {
		log.Fatalf("failed to start mgba thread")
	}
	t.Pause()
	replayer.Reset()
	t.Unpause()
	replayer.Core().GBA().Sync().SetFPSTarget(float32(expectedFPS))

	gameAudioPlayer, err := audioCtx.NewPlayer(av.NewAudioReader(replayer.Core(), replayer.Core().Options().SampleRate))
	if err != nil {
		log.Fatalf("failed to create audio player: %s", err)
	}
	gameAudioPlayer.SetBufferSize(time.Duration(replayer.Core().Options().AudioBuffers+0x4) * time.Second / time.Duration(replayer.Core().Options().SampleRate))
	gameAudioPlayer.Play()

	g := &Game{
		replayer:        replayer,
		vb:              vb,
		gameAudioPlayer: gameAudioPlayer,
		t:               t,
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
