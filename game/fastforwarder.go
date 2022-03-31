package game

import (
	"errors"
	"fmt"
	"log"

	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/input"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/replay"
	"github.com/murkland/bbn6/trapper"
	"github.com/murkland/ringbuf"
)

type fastforwarder struct {
	core  *mgba.Core
	bn6   *bn6.BN6
	state *fastforwarderState
}

type fastforwarderState struct {
	err              error
	localPlayerIndex int
	inputPairs       *ringbuf.RingBuf[[2]input.Input]
	saveState        *mgba.State
	inputConsumed    bool
}

func newFastforwarder(romPath string, bn6 *bn6.BN6) (*fastforwarder, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	ff := &fastforwarder{core, bn6, nil}

	tp := trapper.New(core)

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Pop(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		if ip[0].Tick != ip[1].Tick {
			ff.state.err = fmt.Errorf("p1 tick != p2 tick: %d != %d", ip[0].Tick, ip[1].Tick)
			return
		}

		tick := bn6.InBattleTime(core)
		if ip[0].Tick != int(tick) {
			ff.state.err = fmt.Errorf("tick != in battle time: %d != %d", ip[0].Tick, tick)
			return
		}

		bn6.SetPlayerInputState(core, 0, ip[0].Joyflags, ip[0].CustomScreenState)
		if ip[0].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 0, ip[0].Turn)
			log.Printf("p1 turn committed at tick %d", tick)
		}

		bn6.SetPlayerInputState(core, 1, ip[1].Joyflags, ip[1].CustomScreenState)
		if ip[1].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 1, ip[1].Turn)
			log.Printf("p2 turn committed at tick %d", tick)
		}

		ff.state.inputConsumed = true
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

	core.InstallBeefTrap(tp.BeefHandler)

	core.Reset()

	return ff, nil
}

func (ff *fastforwarder) advanceOne() error {
	// Run until one item is consumed from the input queue.
	ff.state.inputConsumed = false
	ff.state.err = nil
	defer func() {
		ff.state.inputConsumed = false
		ff.state.err = nil
	}()
	for !ff.state.inputConsumed {
		ff.core.RunFrame()
		if ff.state.err != nil {
			return ff.state.err
		}
	}
	return nil
}

func (ff *fastforwarder) advanceMany(rw *replay.Writer, localPlayerIndex int, inputPairs [][2]input.Input) (*mgba.State, error) {
	ff.state = &fastforwarderState{
		localPlayerIndex: localPlayerIndex,
		inputPairs:       ringbuf.New[[2]input.Input](len(inputPairs)),
	}
	defer func() { ff.state = nil }()
	ff.state.inputPairs.Push(inputPairs)

	for ff.state.inputPairs.Used() > 0 {
		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]
		ff.core.SetKeys(mgba.Keys(ip[localPlayerIndex].Joyflags & ^uint16(0xfc00)))
		if err := ff.advanceOne(); err != nil {
			return nil, err
		}
		if err := rw.Write(ff.bn6.RNG2State(ff.core), ip); err != nil {
			return nil, err
		}
	}
	if ff.state.saveState == nil {
		return nil, errors.New("fastforwarder never returned a save state")
	}
	return ff.state.saveState, nil
}

// fastforward fastfowards the state to the new state.
//
// BEWARE: only one thread may call fastforward at a time.
func (ff *fastforwarder) fastforward(state *mgba.State, rw *replay.Writer, localPlayerIndex int, inputPairs [][2]input.Input, lastCommittedRemoteInput input.Input, localPlayerInputsLeft []input.Input) (*mgba.State, *mgba.State, error) {
	if !ff.core.LoadState(state) {
		return nil, nil, errors.New("failed to load state")
	}

	// Run the paired inputs we already have and create the new committed state.
	var committedState *mgba.State
	if len(inputPairs) > 0 {
		newState, err := ff.advanceMany(rw, localPlayerIndex, inputPairs)
		if err != nil {
			return nil, nil, err
		}
		committedState = newState
	} else {
		committedState = state
	}

	// Predict input pairs before fastforwarding dirty state.
	predictedInputPairs := make([][2]input.Input, len(localPlayerInputsLeft))
	for i, inp := range localPlayerInputsLeft {
		predictedInputPairs[i][localPlayerIndex] = inp

		predicted := &predictedInputPairs[i][1-localPlayerIndex]
		predicted.Tick = inp.Tick
		predicted.CustomScreenState = lastCommittedRemoteInput.CustomScreenState
		if lastCommittedRemoteInput.Joyflags&uint16(mgba.KeysA) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysA)
		}
		if lastCommittedRemoteInput.Joyflags&uint16(mgba.KeysB) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysB)
		}
	}

	// Run the local inputs and predict what the remote side did and create the new dirty state.
	var dirtyState *mgba.State
	if len(predictedInputPairs) > 0 {
		newState, err := ff.advanceMany(rw, localPlayerIndex, predictedInputPairs)
		if err != nil {
			return nil, nil, err
		}
		dirtyState = newState
	} else {
		dirtyState = committedState
	}

	return committedState, dirtyState, nil
}
