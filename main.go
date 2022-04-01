package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/Xuanwo/go-locale"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
	_ "github.com/murkland/bbn6/translations"
	"github.com/ncruces/zenity"
	"golang.org/x/exp/maps"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	logFile    = flag.String("log_file", "bbn6.log", "file to log to")
	configPath = flag.String("config_path", "bbn6.toml", "path to config")
	romPath    = flag.String("rom_path", "", "path to rom to start immediately")
)

var version string

func main() {
	flag.Parse()

	lang, err := locale.Detect()
	if err != nil {
		log.Printf("could not detect language, falling back to english: %s", err)
		lang = language.AmericanEnglish
	}
	log.Printf("detected language: %s", lang)
	p := message.NewPrinter(lang)

	var conf config.Config
	confF, err := os.Open(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config doesn't exist, making a new one at: %s", *configPath)
			confF, err = os.Create(*configPath)
			if err != nil {
				log.Fatalf("failed to open config: %s", err)
			}
			conf = config.DefaultConfig
			if err := config.Save(conf, confF); err != nil {
				log.Fatalf("failed to save config: %s", err)
			}
		} else {
			log.Fatalf("failed to open config: %s", err)
		}
	} else {
		conf, err = config.Load(confF)
		if err != nil {
			log.Fatalf("failed to open config: %s", err)
		}
		confF.Close()
	}

	os.MkdirAll("saves", 0o700)
	os.MkdirAll("roms", 0o700)
	os.MkdirAll("replays", 0o700)

	log.Printf("config settings: %+v", conf.ToRaw())

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	log.Printf("welcome to bingus battle network 6 %s", version)

	if *romPath == "" {
		roms, err := os.ReadDir("roms")
		if err != nil {
			log.Fatalf("failed to open roms directory: %s", err)
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

				core.Config().Init("bbn6")
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

		key, err := zenity.List(p.Sprintf("SELECT_ROM"), keys, zenity.Title("bbn6"))
		if err != nil {
			log.Fatalf("failed to select game: %s", err)
		}

		*romPath = filepath.Join("roms", options[key])
	}

	log.Printf("loading rom: %s", *romPath)

	ebiten.SetWindowTitle("bbn6")
	ebiten.SetWindowResizable(true)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)

	g, err := game.New(conf, p, *romPath)
	if err != nil {
		log.Fatalf("failed to start game: %s", err)
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("failed to run game: %s", err)
	}

	g.Finish()
}
