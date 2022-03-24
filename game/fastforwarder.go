package game

import (
	"errors"
	"fmt"

	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/trapper"
	"github.com/murkland/ringbuf"
)

type fastforwarder struct {
	core       *mgba.Core
	inputPairs *ringbuf.RingBuf[[2]Input]
}

func newFastforwarder(romPath string, offsets bn6.Offsets) (*fastforwarder, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	ff := &fastforwarder{core, nil}

	tp := trapper.New(core)
	tp.Add(offsets.A_battle_update__call__battle_copyInputData, func() {
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		var inputPairBuf [1][2]Input
		ff.inputPairs.Pop(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		if ip[0].Tick != ip[1].Tick {
			panic(fmt.Sprintf("p1 tick != p2 tick: %df != %df", ip[0].Tick, ip[1].Tick))
		}

		bn6.SetPlayerInputState(core, 0, ip[0].Joyflags, ip[0].CustomScreenState)
		bn6.SetPlayerInputState(core, 1, ip[1].Joyflags, ip[1].CustomScreenState)

		if ip[0].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 0, ip[0].Turn)
		}

		if ip[1].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 1, ip[1].Turn)
		}
	})
	core.InstallBeefTrap(tp.BeefHandler)

	core.Reset()

	return ff, nil
}

// fastforward fastfowards the state to the new state.
//
// BEWARE: only one thread may call fastforward at a time.
func (ff *fastforwarder) fastforward(state *mgba.State, localPlayerIndex int, inputPairs [][2]Input, localPlayerInputsLeft []Input) (*mgba.State, *mgba.State, error) {
	if !ff.core.LoadState(state) {
		return nil, nil, errors.New("failed to load state")
	}

	// Run the paired inputs we already have and create the new committed state.
	ff.inputPairs = ringbuf.New[[2]Input](len(inputPairs))
	ff.inputPairs.Push(inputPairs)

	for ff.inputPairs.Used() > 0 {
		ff.core.RunFrame()
	}

	committedState := ff.core.SaveState()

	// Run the local inputs and predict what the remote side did and create the new dirty state.
	lastRemoteInput := inputPairs[len(inputPairs)-1][1-localPlayerIndex]

	predictedInputPairs := make([][2]Input, len(localPlayerInputsLeft))
	for i, inp := range localPlayerInputsLeft {
		predictedInputPairs[i][localPlayerIndex] = inp
		inp2 := lastRemoteInput
		inp2.Tick = inp.Tick
		predictedInputPairs[i][1-localPlayerIndex] = inp2
	}

	ff.inputPairs = ringbuf.New[[2]Input](len(localPlayerInputsLeft))
	ff.inputPairs.Push(predictedInputPairs)

	for ff.inputPairs.Used() > 0 {
		ff.core.RunFrame()
	}

	dirtyState := ff.core.SaveState()

	return committedState, dirtyState, nil
}
