package game

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

type InputLog struct {
	f io.WriteCloser
}

func newInputLog(filename string) (*InputLog, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &InputLog{f}, nil
}

func (il *InputLog) WriteInit(playerIndex int, marshaled []byte) error {
	if _, err := fmt.Fprintf(il.f, "init p%d: %s\n", playerIndex+1, hex.EncodeToString(marshaled)); err != nil {
		return err
	}
	return nil
}

func (il *InputLog) Write(rngState uint32, inputPair [2]Input) error {
	p1 := inputPair[0]
	p2 := inputPair[1]

	if _, err := fmt.Fprintf(il.f, "%d: rngstate=%08x p1joyflags=%04x p2joyflags=%04x p1custstate=%d p2custstate=%d\n", p1.Tick, rngState, p1.Joyflags, p2.Joyflags, p1.CustomScreenState, p2.CustomScreenState); err != nil {
		return err
	}

	if p1.Turn != nil {
		if _, err := fmt.Fprintf(il.f, " +p1 turn: %s\n", hex.EncodeToString(p1.Turn)); err != nil {
			return err
		}
	}

	if p2.Turn != nil {
		if _, err := fmt.Fprintf(il.f, " +p2 turn: %s\n", hex.EncodeToString(p2.Turn)); err != nil {
			return err
		}
	}

	return nil
}

func (il *InputLog) Close() error {
	return il.f.Close()
}
