package game

import (
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
	"github.com/murkland/bbn6/mgba"
)

const replayVersion = 0x02
const replayHeader = "TOOT"

type ReplayWriter struct {
	closer io.Closer
	w      io.WriteCloser
}

func newReplayWriter(filename string, gameTitle string) (*ReplayWriter, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	w, err := zstd.NewWriter(f)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write([]byte(replayHeader)); err != nil {
		return nil, err
	}

	if err := binary.Write(w, binary.LittleEndian, uint8(replayVersion)); err != nil {
		return nil, err
	}

	var rawGameTitle [12]byte
	copy(rawGameTitle[:], []byte(gameTitle))
	if _, err := w.Write(rawGameTitle[:]); err != nil {
		return nil, err
	}

	return &ReplayWriter{f, w}, nil
}

func (rw *ReplayWriter) WriteState(playerIndex int, state *mgba.State) error {
	if err := binary.Write(rw.w, binary.LittleEndian, uint8(playerIndex)); err != nil {
		return err
	}

	gs := state.Bytes()
	if err := binary.Write(rw.w, binary.LittleEndian, uint32(len(gs))); err != nil {
		return err
	}
	if _, err := rw.w.Write(gs); err != nil {
		return err
	}

	return nil
}

func (rw *ReplayWriter) WriteInit(playerIndex int, marshaled []byte) error {
	if err := binary.Write(rw.w, binary.LittleEndian, uint8(playerIndex)); err != nil {
		return err
	}

	if len(marshaled) != 0x100 {
		return errors.New("invalid init size")
	}

	if _, err := rw.w.Write(marshaled); err != nil {
		return err
	}

	return nil
}

func (rw *ReplayWriter) Write(rngState uint32, inputPair [2]Input) error {
	p1 := inputPair[0]
	p2 := inputPair[1]

	if err := binary.Write(rw.w, binary.LittleEndian, uint32(p1.Tick)); err != nil {
		return err
	}
	if err := binary.Write(rw.w, binary.LittleEndian, uint32(rngState)); err != nil {
		return err
	}
	if err := binary.Write(rw.w, binary.LittleEndian, uint16(p1.Joyflags)); err != nil {
		return err
	}
	if err := binary.Write(rw.w, binary.LittleEndian, uint8(p1.CustomScreenState)); err != nil {
		return err
	}
	if err := binary.Write(rw.w, binary.LittleEndian, uint16(p2.Joyflags)); err != nil {
		return err
	}
	if err := binary.Write(rw.w, binary.LittleEndian, uint8(p2.CustomScreenState)); err != nil {
		return err
	}

	turnFlags := 0
	if p1.Turn != nil {
		turnFlags |= 0b01
	}
	if p2.Turn != nil {
		turnFlags |= 0b10
	}
	if err := binary.Write(rw.w, binary.LittleEndian, uint8(turnFlags)); err != nil {
		return err
	}

	if p1.Turn != nil {
		if len(p1.Turn) != 0x100 {
			return errors.New("invalid turn size")
		}

		if _, err := rw.w.Write(p1.Turn); err != nil {
			return err
		}
	}

	if p2.Turn != nil {
		if len(p2.Turn) != 0x100 {
			return errors.New("invalid turn size")
		}

		if _, err := rw.w.Write(p2.Turn); err != nil {
			return err
		}
	}

	return nil
}

func (rw *ReplayWriter) Close() error {
	if err := rw.w.Close(); err != nil {
		return err
	}
	if err := rw.closer.Close(); err != nil {
		return err
	}
	return nil
}
