package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
)

var (
	romPath = flag.String("rom_path", "bn6.gba", "path to rom")
)

func main() {
	flag.Parse()

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	replayName := flag.Arg(0)
	f, err := os.Open(replayName)
	if err != nil {
		log.Fatalf("failed to open replay: %s", err)
	}
	defer f.Close()

	replayer, err := game.NewReplayer(*romPath, f)
	if err != nil {
		log.Fatalf("failed to make replayer: %s", err)
	}

	replay := replayer.Replay()

	for i := 0; i < 2; i++ {
		fmt.Fprintf(os.Stdout, "init p%d: %s\n", i+1, hex.EncodeToString(replay.Init[i]))
	}

	for i, inputPair := range replay.InputPairs {
		p1 := inputPair[0]
		p2 := inputPair[1]

		fmt.Fprintf(os.Stdout, "%d: rngstate=%08x p1joyflags=%04x p2joyflags=%04x p1custstate=%d p2custstate=%d\n", p1.Tick, replay.RNGStates[i], p1.Joyflags, p2.Joyflags, p1.CustomScreenState, p2.CustomScreenState)

		if p1.Turn != nil {
			fmt.Fprintf(os.Stdout, " +p1 turn: %s\n", hex.EncodeToString(p1.Turn))
		}

		if p2.Turn != nil {
			fmt.Fprintf(os.Stdout, " +p2 turn: %s\n", hex.EncodeToString(p2.Turn))
		}
	}
}
