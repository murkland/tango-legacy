package main

import (
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/Xuanwo/go-locale"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/tango/bn6"
	"github.com/murkland/tango/config"
	"github.com/murkland/tango/game"
	"github.com/murkland/tango/mgba"

	_ "github.com/murkland/tango/translations"
	"github.com/ncruces/zenity"
	"golang.org/x/exp/maps"
	"golang.org/x/text/message"
)

var (
	configPath = flag.String("config_path", "tango.toml", "path to config")
	romPath    = flag.String("rom_path", "", "path to rom to start immediately")
)

var version string

//go:embed icon.png
var icon []byte

func childMain() {
	img, err := png.Decode(bytes.NewReader(icon))
	if err == nil {
		ebiten.SetWindowIcon([]image.Image{img})
	}

	log.Printf("welcome to tango %s", version)

	lang, _ := locale.Detect()
	lang = message.MatchLanguage(lang.String())
	log.Printf("selected language: %s", lang)
	p := message.NewPrinter(lang)

	conf := config.Default()
	confF, err := os.OpenFile(*configPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		log.Panicf("failed to open config: %s", err)
	} else {
		conf, err = config.Load(confF)
		if err != nil {
			log.Panicf("failed to open config: %s", err)
		}
	}

	if err := confF.Truncate(0); err != nil {
		log.Panicf("failed to truncate config: %s", err)
	}
	if _, err := confF.Seek(0, os.SEEK_SET); err != nil {
		log.Panicf("failed to seek config: %s", err)
	}
	if err := config.Save(conf, confF); err != nil {
		log.Panicf("failed to save config: %s", err)
	}
	confF.Close()

	os.MkdirAll("saves", 0o700)
	os.MkdirAll("roms", 0o700)
	os.MkdirAll("replays", 0o700)

	log.Printf("config settings: %+v", conf)

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	if *romPath == "" {
		roms, err := os.ReadDir("roms")
		if err != nil {
			log.Panicf("failed to open roms directory: %s", err)
		}

		options := map[string]string{}
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

				romTitle := core.GameTitle()

				if bn6 := bn6.Load(romTitle); bn6 == nil {
					return errors.New("unsupported rom")
				}

				options[fmt.Sprintf("%s: %s", dirent.Name(), romTitle)] = dirent.Name()

				return nil
			}(); err != nil {
				continue
			}
		}
		keys := maps.Keys(options)
		sort.Strings(keys)

		key, err := zenity.List(p.Sprintf("SELECT_ROM"), keys, zenity.Title("tango"))
		if err != nil {
			log.Panicf("failed to select game: %s", err)
		}

		*romPath = filepath.Join("roms", options[key])
	}

	log.Printf("loading rom: %s", *romPath)

	ebiten.SetWindowTitle("tango")
	ebiten.SetWindowResizable(true)
	ebiten.SetRunnableOnUnfocused(true)

	g, err := game.New(conf, p, *romPath)
	if err != nil {
		log.Panicf("failed to start game: %s", err)
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Panicf("failed to run game: %s", err)
	}

	g.Finish()
}
