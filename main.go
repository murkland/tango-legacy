package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
	"github.com/ncruces/zenity"
	"golang.org/x/exp/maps"
)

var (
	child      = flag.Bool("child", false, "is this the child process?")
	logFile    = flag.String("log_file", "bbn6.log", "file to log to")
	configPath = flag.String("config_path", "bbn6.toml", "path to config")
	romPath    = flag.String("rom_path", "", "path to rom to start immediately")
)

var commitHash string

func main() {
	flag.Parse()
	if *child {
		childMain()
		return
	}

	runtime.LockOSThread()

	logF, err := os.Create(*logFile)
	if err != nil {
		log.Fatalf("failed to open log file: %s", err)
	}
	defer logF.Close()

	if err := (&exec.Cmd{
		Path:   os.Args[0],
		Args:   append(os.Args, "-child"),
		Stdout: os.Stdout,
		Stderr: io.MultiWriter(os.Stderr, logF),
	}).Run(); err != nil {
		log.Printf("child exited with %s", err)
	}
	fmt.Printf("press any key to continue...")
	fmt.Scanln()
}

func childMain() {
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

	log.Printf("config settings: %+v", conf.ToRaw())

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	ctx := context.Background()

	log.Printf("welcome to bingus battle network 6. commit hash = %s", commitHash)

	if *romPath == "" {
		roms, err := os.ReadDir("roms")
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				zenity.Error("Could not find a directory named \"roms\" in your bbn6 folder.\n\nPlease create one and put all your .gba and .sav files in it and run bbn6 again.", zenity.Title("bbn6"))
				return
			}
			log.Fatalf("failed to open roms directory: %s", err)
		}

		options := map[string]string{}
		for _, dirent := range roms {
			if err := func() error {
				path := filepath.Join("roms", dirent.Name())
				core, err := mgba.FindCore(path)
				if err != nil {
					return err
				}
				defer core.Close()

				if err := core.LoadFile(path); err != nil {
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

		selection, err := zenity.List("Select a game to start:", keys, zenity.Title("bbn6"))
		if err != nil {
			log.Fatalf("failed to select game: %s", err)
		}

		*romPath = filepath.Join("roms", options[selection])
	}

	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowTitle("bbn6")
	ebiten.SetMaxTPS(ebiten.UncappedTPS)
	ebiten.SetWindowResizable(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	g, err := game.New(conf, *romPath)
	if err != nil {
		log.Fatalf("failed to start game: %s", err)
	}

	go func() {
		if err := g.RunBackgroundTasks(ctx); err != nil {
			log.Fatalf("error running background tasks: %s", err)
		}
	}()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("failed to run mgba: %s", err)
	}

	g.Finish()
}
