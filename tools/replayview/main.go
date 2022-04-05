package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/murkland/tango/av"
	"github.com/murkland/tango/game"
	"github.com/murkland/tango/mgba"
	"github.com/murkland/tango/replay"
	"github.com/ncruces/zenity"
)

var (
	romPath = flag.String("rom_path", "bn6.gba", "path to rom")
)

type Game struct {
	replayer *game.Replayer

	vb      *av.VideoBuffer
	vbPixMu sync.Mutex
	vbPix   []byte

	fbuf *ebiten.Image

	gameAudioPlayer *audio.Player

	t *mgba.Thread
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
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

func (g *Game) scaleFactor(bounds image.Rectangle) int {
	w, h := g.replayer.Core().DesiredVideoDimensions()
	k := bounds.Dx() / w
	if s := bounds.Dy() / h; s < k {
		k = s
	}
	return k
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.vbPixMu.Lock()
	defer g.vbPixMu.Unlock()

	k := g.scaleFactor(screen.Bounds())
	opts := &ebiten.DrawImageOptions{}
	w, h := g.replayer.Core().DesiredVideoDimensions()
	opts.GeoM.Scale(float64(k), float64(k))
	opts.GeoM.Translate(float64((screen.Bounds().Dx()-w*k)/2), float64((screen.Bounds().Dy()-h*k)/2))
	g.fbuf.ReplacePixels(g.vbPix)
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
			log.Panicf("failed to prompt for replay: %s", err)
		}
		replayName = fn
	}

	f, err := os.Open(replayName)
	if err != nil {
		log.Panicf("failed to open replay: %s", err)
	}
	defer f.Close()

	r, err := replay.Unmarshal(f)
	if err != nil {
		log.Panicf("failed to open replay: %s", err)
	}

	roms, err := os.ReadDir("roms")
	if err != nil {
		log.Panicf("failed to open roms directory: %s", err)
	}

	var romPath string
	for _, dirent := range roms {
		path := filepath.Join("roms", dirent.Name())

		if err := func() error {
			core, err := mgba.NewGBACore()
			if err != nil {
				return err
			}
			defer core.Close()

			core.Config().Init("tango")
			core.Config().Load()

			vf := mgba.OpenVF(path, os.O_RDONLY)
			if vf == nil {
				return errors.New("failed to open file")
			}

			if err := core.LoadROM(vf); err != nil {
				return err
			}

			if r.State.ROMTitle != core.GameTitle() {
				return fmt.Errorf("rom title doesn't match: %s != %s", r.State.ROMTitle, core.GameTitle())
			}

			if r.State.ROMCRC32 != core.CRC32() {
				return fmt.Errorf("crc32 doesn't match: %08x != %08x", r.State.ROMCRC32, core.CRC32())
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
		log.Panicf("failed find eligible rom")
	}

	replayer, err := game.NewReplayer(romPath, r)
	if err != nil {
		log.Panicf("failed to make replayer: %s", err)
	}

	audioCtx := audio.NewContext(replayer.Core().Options().SampleRate)

	width, height := replayer.Core().DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)
	ebiten.SetWindowSize(width*3, height*3)

	replayer.Core().SetVideoBuffer(vb.Pointer(), width)

	gameAudioPlayer, err := audioCtx.NewPlayer(av.NewRubberyAudioReader(replayer.Core(), replayer.Core().Options().SampleRate))
	if err != nil {
		log.Panicf("failed to create audio player: %s", err)
	}
	gameAudioPlayer.SetBufferSize(time.Duration(replayer.Core().AudioBufferSize()+1) * time.Second / time.Duration(replayer.Core().Options().SampleRate))
	gameAudioPlayer.Play()

	g := &Game{
		replayer:        replayer,
		vb:              vb,
		vbPix:           make([]byte, width*height*4),
		fbuf:            ebiten.NewImage(width, height),
		gameAudioPlayer: gameAudioPlayer,
	}

	g.t = mgba.NewThread(replayer.Core())
	g.t.SetFrameCallback(func() {
		g.vbPixMu.Lock()
		defer g.vbPixMu.Unlock()
		copy(g.vbPix, g.vb.Pix())
	})

	if !g.t.Start() {
		log.Panicf("failed to start mgba thread")
	}
	g.t.Pause()
	replayer.Reset()
	g.t.Unpause()
	replayer.Core().GBA().Sync().SetFPSTarget(float32(expectedFPS))

	ebiten.SetWindowTitle("tango replayview")
	ebiten.SetWindowResizable(true)
	ebiten.SetRunnableOnUnfocused(true)

	if err := ebiten.RunGame(g); err != nil {
		log.Panicf("failed to run mgba: %s", err)
	}
}
