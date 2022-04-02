package game

import (
	"errors"
	"fmt"
	"log"

	"github.com/murkland/ringbuf"
	"github.com/murkland/tango/bn6"
	"github.com/murkland/tango/input"
	"github.com/murkland/tango/mgba"
	"github.com/murkland/tango/replay"
	"github.com/murkland/tango/trapper"
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
	predicting       bool
}

func newFastforwarder(romPath string, bn6 *bn6.BN6) (*fastforwarder, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	ff := &fastforwarder{core, bn6, nil}

	tp := trapper.New(core)

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
			ff.state.err = fmt.Errorf("p1 tick != p2 tick: %d != %d", ip[0].LocalTick, ip[1].LocalTick)
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

func (ff *fastforwarder) applyInputs(state *mgba.State, rw *replay.Writer, localPlayerIndex int, inputPairs [][2]input.Input) (*mgba.State, error) {
	ff.state = &fastforwarderState{
		saveState:        state,
		localPlayerIndex: localPlayerIndex,
		inputPairs:       ringbuf.New[[2]input.Input](len(inputPairs)),
		predicting:       rw == nil,
	}
	defer func() {
		ff.state = nil
	}()
	ff.state.inputPairs.Push(inputPairs)

	if !ff.core.LoadState(ff.state.saveState) {
		return nil, errors.New("failed to load state")
	}

	for ff.state.inputPairs.Used() > 0 {
		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		ff.state.err = nil
		for qlen := ff.state.inputPairs.Used(); ff.state.inputPairs.Used() == qlen; {
			ff.core.RunFrame()
			if ff.state.err != nil {
				return nil, ff.state.err
			}
		}

		if rw != nil {
			if err := rw.Write(ff.bn6.RNG2State(ff.core), ip); err != nil {
				return nil, err
			}
		}
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
	committedState, err := ff.applyInputs(state, rw, localPlayerIndex, inputPairs)
	if err != nil {
		return nil, nil, err
	}

	// Predict input pairs before fastforwarding dirty state.
	predictedInputPairs := make([][2]input.Input, len(localPlayerInputsLeft))
	for i, inp := range localPlayerInputsLeft {
		predictedInputPairs[i][localPlayerIndex] = inp

		predicted := &predictedInputPairs[i][1-localPlayerIndex]
		predicted.LocalTick = inp.LocalTick
		predicted.LastCommittedRemoteTick = inp.LastCommittedRemoteTick
		predicted.CustomScreenState = lastCommittedRemoteInput.CustomScreenState
		if lastCommittedRemoteInput.Joyflags&uint16(mgba.KeysA) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysA)
		}
		if lastCommittedRemoteInput.Joyflags&uint16(mgba.KeysB) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysB)
		}
	}

	// Run the local inputs and predict what the remote side did and create the new dirty state.
	dirtyState, err := ff.applyInputs(committedState, nil, localPlayerIndex, predictedInputPairs)
	if err != nil {
		return nil, nil, err
	}

	return committedState, dirtyState, nil
}
