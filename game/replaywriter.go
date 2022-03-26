package game

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/murkland/bbn6/mgba"
)

const replayVersion = 0x01
const replayHeader = "TOOT"

type ReplayWriter struct {
	f io.WriteCloser
}

func newReplayWriter(filename string) (*ReplayWriter, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	if err := binary.Write(f, binary.LittleEndian, uint8(replayVersion)); err != nil {
		return nil, err
	}

	return &ReplayWriter{f}, nil
}

func (rw *ReplayWriter) WriteState(playerIndex int, state *mgba.State) error {
	if _, err := rw.f.Write([]byte(replayHeader)); err != nil {
		return err
	}

	if err := binary.Write(rw.f, binary.LittleEndian, uint8(playerIndex)); err != nil {
		return err
	}

	gs := state.Bytes()
	if err := binary.Write(rw.f, binary.LittleEndian, uint32(len(gs))); err != nil {
		return err
	}
	if _, err := rw.f.Write(gs); err != nil {
		return err
	}

	return nil
}

func (rw *ReplayWriter) WriteInit(playerIndex int, marshaled []byte) error {
	if _, err := rw.f.Write(marshaled); err != nil {
		return err
	}

	return nil
}

func (rw *ReplayWriter) Write(rngState uint32, inputPair [2]Input) error {
	p1 := inputPair[0]
	p2 := inputPair[1]

	if err := binary.Write(rw.f, binary.LittleEndian, uint32(p1.Tick)); err != nil {
		return err
	}
	if err := binary.Write(rw.f, binary.LittleEndian, uint32(rngState)); err != nil {
		return err
	}
	if err := binary.Write(rw.f, binary.LittleEndian, uint16(p1.Joyflags)); err != nil {
		return err
	}
	if err := binary.Write(rw.f, binary.LittleEndian, uint8(p1.CustomScreenState)); err != nil {
		return err
	}
	if err := binary.Write(rw.f, binary.LittleEndian, uint16(p2.Joyflags)); err != nil {
		return err
	}
	if err := binary.Write(rw.f, binary.LittleEndian, uint8(p2.CustomScreenState)); err != nil {
		return err
	}

	turnFlags := 0
	if p1.Turn != nil {
		turnFlags |= 0b01
	}
	if p2.Turn != nil {
		turnFlags |= 0b10
	}
	if err := binary.Write(rw.f, binary.LittleEndian, uint8(turnFlags)); err != nil {
		return err
	}

	if p1.Turn != nil {
		if _, err := rw.f.Write(p1.Turn); err != nil {
			return err
		}
	}

	if p2.Turn != nil {
		if _, err := rw.f.Write(p2.Turn); err != nil {
			return err
		}
	}

	return nil
}

func (rw *ReplayWriter) Close() error {
	return rw.f.Close()
}
