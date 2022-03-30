package replay

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/klauspost/compress/zstd"
	"github.com/murkland/bbn6/input"
	"github.com/murkland/bbn6/mgba"
)

type Replay struct {
	State            *mgba.State
	LocalPlayerIndex int
	Init             [2][]byte
	InputPairs       [][2]input.Input
	RNGStates        []uint32
}

// Marshaled replay format is:
//
// header:
// u8[4]: TOOT
// u8: replay version
// u8: local player index
// u32: state size
// state size: state
//
// init (two of them):
// u8: player index
// init size: init
//
// inputs:
// u32: tick
// u32: rng2state
// u16: p1joyflags
// u8: p1customstate
// u16: p2joyflags
// u8: p2customstate
// u8: turn flags (0b00 = nobody, 0b01 = p1, 0b10 = p2, 0b11 = p1 and p2)
// turn size: turn data
func Unmarshal(r io.Reader) (*Replay, error) {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}

	// read header
	var header [4]byte
	if _, err := io.ReadFull(zr, header[:]); err != nil {
		return nil, err
	}

	if string(header[:]) != replayHeader {
		return nil, fmt.Errorf("invalid format")
	}

	var version uint8
	if err := binary.Read(zr, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version != replayVersion {
		return nil, fmt.Errorf("unsupported replay version: %02x vs %02x", version, replayVersion)
	}

	var localPlayerIndex uint8
	if err := binary.Read(zr, binary.LittleEndian, &localPlayerIndex); err != nil {
		return nil, err
	}

	var stateSize uint32
	if err := binary.Read(zr, binary.LittleEndian, &stateSize); err != nil {
		return nil, err
	}

	stateBytes := make([]byte, int(stateSize))
	if _, err := io.ReadFull(zr, stateBytes); err != nil {
		return nil, err
	}
	state := mgba.StateFromBytes(stateBytes)

	// read inits
	var init [2][]byte
	for i := 0; i < 2; i++ {
		var playerIndex uint8
		if err := binary.Read(zr, binary.LittleEndian, &playerIndex); err != nil {
			return nil, err
		}

		var marshaled [0x100]byte
		if _, err := io.ReadFull(zr, marshaled[:]); err != nil {
			return nil, err
		}

		init[playerIndex] = marshaled[:]
	}

	// read inputs
	var inputPairs [][2]input.Input
	var rngStates []uint32
	for {
		var tick uint32
		if err := binary.Read(zr, binary.LittleEndian, &tick); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}

		// we don't do anything with this, actually...
		var rngState uint32
		if err := binary.Read(zr, binary.LittleEndian, &rngState); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}
		rngStates = append(rngStates, rngState)

		var inputPair [2]input.Input
		inputPair[0].Tick = int(tick)
		inputPair[1].Tick = int(tick)

		var p1Joyflags uint16
		if err := binary.Read(zr, binary.LittleEndian, &p1Joyflags); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}
		inputPair[0].Joyflags = p1Joyflags

		var p1CustomScreenState uint8
		if err := binary.Read(zr, binary.LittleEndian, &p1CustomScreenState); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}
		inputPair[0].CustomScreenState = p1CustomScreenState

		var p2Joyflags uint16
		if err := binary.Read(zr, binary.LittleEndian, &p2Joyflags); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}
		inputPair[1].Joyflags = p2Joyflags

		var p2CustomScreenState uint8
		if err := binary.Read(zr, binary.LittleEndian, &p2CustomScreenState); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}
		inputPair[1].CustomScreenState = p2CustomScreenState

		var turnFlags uint8
		if err := binary.Read(zr, binary.LittleEndian, &turnFlags); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				log.Printf("replay was truncated")
				break
			}
			return nil, err
		}

		if turnFlags&0b01 != 0 {
			var turn [0x100]byte
			if _, err := io.ReadFull(zr, turn[:]); err != nil {
				if errors.Is(err, io.ErrUnexpectedEOF) {
					log.Printf("replay was truncated")
					break
				}
				return nil, err
			}
			inputPair[0].Turn = turn[:]
		}

		if turnFlags&0b10 != 0 {
			var turn [0x100]byte
			if _, err := io.ReadFull(zr, turn[:]); err != nil {
				if errors.Is(err, io.ErrUnexpectedEOF) {
					log.Printf("replay was truncated")
					break
				}
				return nil, err
			}
			inputPair[1].Turn = turn[:]
		}

		inputPairs = append(inputPairs, inputPair)
	}

	return &Replay{
		State:            state,
		LocalPlayerIndex: int(localPlayerIndex),
		Init:             init,
		InputPairs:       inputPairs,
		RNGStates:        rngStates,
	}, nil
}
