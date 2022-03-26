package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

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
	State            *mgba.State
	LocalPlayerIndex int
	Init             [2][]byte
	InputPairs       [][2]Input
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
func deserializeReplay(r io.Reader) (*Replay, error) {
	// read header
	var header [4]byte
	if _, err := r.Read(header[:]); err != nil {
		return nil, err
	}

	if string(header[:]) != replayHeader {
		return nil, fmt.Errorf("invalid format")
	}

	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version != replayVersion {
		return nil, fmt.Errorf("unsupported replay version: %02x vs %02x", version, replayVersion)
	}

	var localPlayerIndex uint8
	if err := binary.Read(r, binary.LittleEndian, &localPlayerIndex); err != nil {
		return nil, err
	}

	var stateSize uint32
	if err := binary.Read(r, binary.LittleEndian, &stateSize); err != nil {
		return nil, err
	}

	stateBytes := make([]byte, int(stateSize))
	if _, err := r.Read(stateBytes); err != nil {
		return nil, err
	}
	state := mgba.StateFromBytes(stateBytes)

	// read inits
	var init [2][]byte
	for i := 0; i < 2; i++ {
		var playerIndex uint8
		if err := binary.Read(r, binary.LittleEndian, &playerIndex); err != nil {
			return nil, err
		}

		var marshaled [0x100]byte
		if _, err := r.Read(marshaled[:]); err != nil {
			return nil, err
		}

		init[playerIndex] = marshaled[:]
	}

	// read inputs
	var inputPairs [][2]Input
	for {
		var tick uint32
		if err := binary.Read(r, binary.LittleEndian, &tick); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		// we don't do anything with this, actually...
		var rngState uint32
		if err := binary.Read(r, binary.LittleEndian, &rngState); err != nil {
			return nil, err
		}

		var inputPair [2]Input
		inputPair[0].Tick = int(tick)
		inputPair[1].Tick = int(tick)

		var p1Joyflags uint16
		if err := binary.Read(r, binary.LittleEndian, &p1Joyflags); err != nil {
			return nil, err
		}
		inputPair[0].Joyflags = p1Joyflags

		var p1CustomScreenState uint8
		if err := binary.Read(r, binary.LittleEndian, &p1CustomScreenState); err != nil {
			return nil, err
		}
		inputPair[0].CustomScreenState = p1CustomScreenState

		var p2Joyflags uint16
		if err := binary.Read(r, binary.LittleEndian, &p2Joyflags); err != nil {
			return nil, err
		}
		inputPair[1].Joyflags = p2Joyflags

		var p2CustomScreenState uint8
		if err := binary.Read(r, binary.LittleEndian, &p2CustomScreenState); err != nil {
			return nil, err
		}
		inputPair[1].CustomScreenState = p2CustomScreenState

		var turnFlags uint8
		if err := binary.Read(r, binary.LittleEndian, &turnFlags); err != nil {
			return nil, err
		}

		if turnFlags&0b01 != 0 {
			var turn [0x100]byte
			if _, err := r.Read(turn[:]); err != nil {
				return nil, err
			}
			inputPair[0].Turn = turn[:]
		}

		if turnFlags&0b10 != 0 {
			var turn [0x100]byte
			if _, err := r.Read(turn[:]); err != nil {
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
	}, nil
}

func newReplayer(romPath string, bn6 *bn6.BN6, replayR io.Reader) (*Replayer, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	replay, err := deserializeReplay(replayR)
	if err != nil {
		return nil, err
	}

	rp := &Replayer{core, bn6, replay, nil}

	tp := trapper.New(core)

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		var inputPairBuf [1][2]Input
		rp.currentInputPairs.Pop(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		if ip[0].Tick != ip[1].Tick {
			panic(fmt.Sprintf("p1 tick != p2 tick: %d != %d", ip[0].Tick, ip[1].Tick))
		}

		bn6.SetPlayerInputState(core, 0, ip[0].Joyflags, ip[0].CustomScreenState)
		if ip[0].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 0, ip[0].Turn)
		}

		bn6.SetPlayerInputState(core, 1, ip[1].Joyflags, ip[1].CustomScreenState)
		if ip[1].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 1, ip[1].Turn)
		}
	})

	tp.Add(bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		core.GBA().SetRegister(0, uint32(rp.replay.LocalPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_link_isP2__ret, func() {
		core.GBA().SetRegister(0, uint32(rp.replay.LocalPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_endBattle__entry, func() {
		rp.Reset()
	})

	core.InstallBeefTrap(tp.BeefHandler)

	rp.Reset()

	return rp, nil
}
