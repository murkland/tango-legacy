package game

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/murkland/bbn6/mgba"
)

type ReplayWriter struct {
	f io.WriteCloser
}

func newReplayWriter(filename string) (*ReplayWriter, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &ReplayWriter{f}, nil
}

func (rw *ReplayWriter) WriteState(playerIndex int, state *mgba.State) error {
	return nil
}

func (rw *ReplayWriter) WriteInit(playerIndex int, marshaled []byte) error {
	if _, err := fmt.Fprintf(rw.f, "init p%d: %s\n", playerIndex+1, hex.EncodeToString(marshaled)); err != nil {
		return err
	}
	return nil
}

func (rw *ReplayWriter) Write(rngState uint32, inputPair [2]Input) error {
	p1 := inputPair[0]
	p2 := inputPair[1]

	if _, err := fmt.Fprintf(rw.f, "%d: rngstate=%08x p1joyflags=%04x p2joyflags=%04x p1custstate=%d p2custstate=%d\n", p1.Tick, rngState, p1.Joyflags, p2.Joyflags, p1.CustomScreenState, p2.CustomScreenState); err != nil {
		return err
	}

	if p1.Turn != nil {
		if _, err := fmt.Fprintf(rw.f, " +p1 turn: %s\n", hex.EncodeToString(p1.Turn)); err != nil {
			return err
		}
	}

	if p2.Turn != nil {
		if _, err := fmt.Fprintf(rw.f, " +p2 turn: %s\n", hex.EncodeToString(p2.Turn)); err != nil {
			return err
		}
	}

	return nil
}

func (rw *ReplayWriter) Close() error {
	return rw.f.Close()
}
