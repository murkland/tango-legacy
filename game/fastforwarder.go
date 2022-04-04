package game

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/murkland/ringbuf"
	"github.com/murkland/tango/bn6"
	"github.com/murkland/tango/input"
	"github.com/murkland/tango/mgba"
	"github.com/murkland/tango/replay"
)

type Fastforwarder struct {
	core                    *mgba.Core
	bn6                     *bn6.BN6
	state                   *fastforwarderState
	lastFastforwardDuration time.Duration
}

type fastforwarderState struct {
	err              error
	localPlayerIndex int
	inputPairs       *ringbuf.RingBuf[[2]input.Input]
	saveState        *mgba.State
	tick             int
	predicting       bool
}

func NewFastforwarder(romPath string, bn6 *bn6.BN6) (*Fastforwarder, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	ff := &Fastforwarder{core, bn6, nil, 0}

	tp := mgba.NewTrapper(core)

	tp.Add(bn6.Offsets.ROM.A_main__readJoyflags, func() {
		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]
		core.GBA().SetRegister(4, uint32(ip[ff.state.localPlayerIndex].Joyflags))
	})

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Pop(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		if ip[0].LocalTick != ip[1].LocalTick {
			ff.state.err = fmt.Errorf("p1 tick != p2 tick (predicting = %t): %d != %d", ff.state.predicting, ip[0].LocalTick, ip[1].LocalTick)
			return
		}

		if ip[0].LocalTick != ff.state.tick {
			ff.state.err = fmt.Errorf("input tick != state tick (predicting = %t): %d != %d", ff.state.predicting, ip[0].LocalTick, ff.state.tick)
			return
		}

		bn6.SetPlayerInputState(core, 0, ip[0].Joyflags, ip[0].CustomScreenState)
		if ip[0].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 0, ip[0].Turn)
			if !ff.state.predicting {
				log.Printf("p1 turn committed at tick %d", ip[0].LocalTick)
			}
		}

		bn6.SetPlayerInputState(core, 1, ip[1].Joyflags, ip[1].CustomScreenState)
		if ip[1].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 1, ip[1].Turn)
			if !ff.state.predicting {
				log.Printf("p2 turn committed at tick %d", ip[1].LocalTick)
			}
		}

		ff.state.tick++

		if ff.state.inputPairs.Used() == 0 {
			ff.state.saveState = core.SaveState()
		}
	})

	tp.Add(bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		core.GBA().SetRegister(0, uint32(ff.state.localPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_link_isP2__ret, func() {
		core.GBA().SetRegister(0, uint32(ff.state.localPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(bn6.Offsets.ROM.A_getCopyDataInputState__ret, func() {
		core.GBA().SetRegister(0, 2)
	})

	tp.Attach(core.GBA())

	core.Reset()

	return ff, nil
}

func (ff *Fastforwarder) applyInputs(tick int, state *mgba.State, rw *replay.Writer, localPlayerIndex int, inputPairs [][2]input.Input) (int, *mgba.State, error) {
	ff.state = &fastforwarderState{
		saveState:        state,
		tick:             tick,
		localPlayerIndex: localPlayerIndex,
		inputPairs:       ringbuf.New[[2]input.Input](len(inputPairs)),
		predicting:       rw == nil,
	}
	defer func() {
		ff.state = nil
	}()
	ff.state.inputPairs.Push(inputPairs)

	for ff.state.inputPairs.Used() > 0 {
		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		ff.state.err = nil
		qlen := ff.state.inputPairs.Used()
		for ff.state.inputPairs.Used() == qlen {
			ff.core.RunFrame()
			if ff.state.err != nil {
				return ff.state.tick, nil, ff.state.err
			}
		}
		if ff.state.inputPairs.Used() != qlen-1 {
			return ff.state.tick, nil, fmt.Errorf("queue length decreased by more than 1: %d -> %d", qlen, ff.state.inputPairs.Used())
		}

		if rw != nil {
			if err := rw.Write(ff.bn6.RNG2State(ff.core), ip); err != nil {
				return ff.state.tick, nil, err
			}
		}
	}

	return ff.state.tick, ff.state.saveState, nil
}

// fastforward fastfowards the state to the new state.
//
// BEWARE: only one thread may call fastforward at a time.
func (ff *Fastforwarder) Fastforward(tick int, state *mgba.State, rw *replay.Writer, localPlayerIndex int, inputPairs [][2]input.Input, lastCommittedRemoteInput input.Input, localPlayerInputsLeft []input.Input) (int, *mgba.State, *mgba.State, error) {
	startTime := time.Now()

	if !ff.core.LoadState(state) {
		return tick, nil, nil, errors.New("failed to load state")
	}

	// Run the paired inputs we already have and create the new committed state.
	committedTick, committedState, err := ff.applyInputs(tick, state, rw, localPlayerIndex, inputPairs)
	if err != nil {
		return tick, nil, nil, err
	}

	if !ff.core.LoadState(committedState) {
		return tick, nil, nil, errors.New("failed to load committed state")
	}

	// Predict input pairs before fastforwarding dirty state.
	predictedInputPairs := make([][2]input.Input, len(localPlayerInputsLeft))
	for i, inp := range localPlayerInputsLeft {
		predictedInputPairs[i][localPlayerIndex] = inp

		predicted := &predictedInputPairs[i][1-localPlayerIndex]
		predicted.LocalTick = inp.LocalTick
		predicted.RemoteTick = inp.RemoteTick
		predicted.CustomScreenState = lastCommittedRemoteInput.CustomScreenState
		if lastCommittedRemoteInput.Joyflags&uint16(mgba.KeysA) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysA)
		}
		if lastCommittedRemoteInput.Joyflags&uint16(mgba.KeysB) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysB)
		}
	}

	// Run the local inputs and predict what the remote side did and create the new dirty state.
	_, dirtyState, err := ff.applyInputs(committedTick, committedState, nil, localPlayerIndex, predictedInputPairs)
	if err != nil {
		return committedTick, nil, nil, err
	}

	ff.lastFastforwardDuration = time.Now().Sub(startTime)

	return committedTick, committedState, dirtyState, nil
}
