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
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
)

var (
	child      = flag.Bool("child", false, "is this the child process?")
	logFile    = flag.String("log_file", "bbn6.log", "file to log to")
	configPath = flag.String("config_path", "bbn6.toml", "path to config")
	romPath    = flag.String("rom_path", "bn6.gba", "path to rom")
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
	ctx := context.Background()

	log.Printf("welcome to bingus battle network 6. commit hash = %s", commitHash)

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
