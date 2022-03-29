package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/klauspost/compress/zstd"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/trapper"
	"github.com/murkland/ringbuf"
)

type Replayer struct {
	core *mgba.Core
	bn6  *bn6.BN6

	replay *Replay

	currentInputPairs *ringbuf.RingBuf[[2]Input]
}

type Replay struct {
	ROMTitle         string
	ROMCRC32         uint32
	State            *mgba.State
	LocalPlayerIndex int
	Init             [2][]byte
	InputPairs       [][2]Input
	RNGStates        []uint32
}

func (rp *Replayer) Reset() {
	rp.currentInputPairs = ringbuf.New[[2]Input](len(rp.replay.InputPairs))
	rp.currentInputPairs.Push(rp.replay.InputPairs)
	rp.core.LoadState(rp.replay.State)
	rp.bn6.SetPlayerMarshaledBattleState(rp.core, 0, rp.replay.Init[0])
	rp.bn6.SetPlayerMarshaledBattleState(rp.core, 1, rp.replay.Init[1])
}

// Serialized replay format is:
//
// header:
// u8[4]: TOOT
// u8: replay version
// u8[12]: game title
// u32: game crc32
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
func DeserializeReplay(r io.Reader) (*Replay, error) {
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

	var titleRaw [12]byte
	if _, err := io.ReadFull(zr, titleRaw[:]); err != nil {
		return nil, err
	}
	gameTitle := string(bytes.TrimRight(titleRaw[:], "\x00"))

	var crc32 uint32
	if err := binary.Read(zr, binary.LittleEndian, &crc32); err != nil {
		return nil, err
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
	var inputPairs [][2]Input
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

		var inputPair [2]Input
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
		ROMTitle:         gameTitle,
		ROMCRC32:         crc32,
		State:            state,
		LocalPlayerIndex: int(localPlayerIndex),
		Init:             init,
		InputPairs:       inputPairs,
		RNGStates:        rngStates,
	}, nil
}

func (rp *Replayer) Core() *mgba.Core {
	return rp.core
}

func (rp *Replayer) PeekLocalJoyflags() uint16 {
	if rp.currentInputPairs.Used() == 0 {
		return 0xfc00
	}

	var inputPairBuf [1][2]Input
	rp.currentInputPairs.Peek(inputPairBuf[:], 0)
	ip := inputPairBuf[0]
	return ip[rp.replay.LocalPlayerIndex].Joyflags
}

func NewReplayer(romPath string, replay *Replay) (*Replayer, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	bn6 := bn6.Load(core.GameTitle())
	if bn6 == nil {
		return nil, fmt.Errorf("unsupported game: %s", core.GameTitle())
	}

	rp := &Replayer{core, bn6, replay, nil}

	tp := trapper.New(rp.core)

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		rp.core.GBA().SetRegister(0, 0)
		rp.core.GBA().SetRegister(15, rp.core.GBA().Register(15)+4)
		rp.core.GBA().ThumbWritePC()

		var inputPairBuf [1][2]Input
		rp.currentInputPairs.Pop(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		bn6.SetPlayerInputState(rp.core, 0, ip[0].Joyflags, ip[0].CustomScreenState)
		if ip[0].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(rp.core, 0, ip[0].Turn)
		}

		bn6.SetPlayerInputState(rp.core, 1, ip[1].Joyflags, ip[1].CustomScreenState)
		if ip[1].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(rp.core, 1, ip[1].Turn)
		}
	})

	tp.Add(bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		rp.core.GBA().SetRegister(0, uint32(rp.replay.LocalPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_link_isP2__ret, func() {
		rp.core.GBA().SetRegister(0, uint32(rp.replay.LocalPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		rp.core.GBA().SetRegister(15, rp.core.GBA().Register(15)+4)
		rp.core.GBA().ThumbWritePC()
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_endBattle__entry, func() {
		rp.Reset()
	})

	rp.core.InstallBeefTrap(tp.BeefHandler)

	return rp, nil
}

func (r *Replayer) Replay() *Replay {
	return r.replay
}
