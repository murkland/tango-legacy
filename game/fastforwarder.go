package game

import (
	"errors"
	"fmt"
	"log"

	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/trapper"
	"github.com/murkland/ringbuf"
)

type fastforwarder struct {
	core *mgba.Core
	bn6  *bn6.BN6

	localPlayerIndex int
	inputPairs       *ringbuf.RingBuf[[2]Input]
	state            *mgba.State
	tick             int
}

func newFastforwarder(romPath string, bn6 *bn6.BN6) (*fastforwarder, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	ff := &fastforwarder{core, bn6, 0, nil, nil, 0}

	tp := trapper.New(core)

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		ff.tick++

		var inputPairBuf [1][2]Input
		ff.inputPairs.Pop(inputPairBuf[:], 0)
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

		if ff.inputPairs.Used() == 0 {
			ff.state = core.SaveState()
		}
	})

	tp.Add(bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		core.GBA().SetRegister(0, uint32(ff.localPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_link_isP2__ret, func() {
		core.GBA().SetRegister(0, uint32(ff.localPlayerIndex))
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()
	})

	core.InstallBeefTrap(tp.BeefHandler)

	core.Reset()

	return ff, nil
}

func (ff *fastforwarder) advanceOne() {
	currentTick := ff.tick
	framesAdvanced := 0
	for ff.tick == currentTick {
		ff.core.RunFrame()
		framesAdvanced++
	}
	if framesAdvanced > 2 {
		log.Printf("game took a long time (%d frames) to process one input on tick %d", framesAdvanced, currentTick)
	}
}

// fastforward fastfowards the state to the new state.
//
// BEWARE: only one thread may call fastforward at a time.
func (ff *fastforwarder) fastforward(state *mgba.State, il *InputLog, localPlayerIndex int, inputPairs [][2]Input, localPlayerInputsLeft []Input) (*mgba.State, *mgba.State, error) {
	if !ff.core.LoadState(state) {
		return nil, nil, errors.New("failed to load state")
	}

	ff.state = nil
	ff.localPlayerIndex = localPlayerIndex

	// Run the paired inputs we already have and create the new committed state.
	ff.inputPairs = ringbuf.New[[2]Input](len(inputPairs))
	ff.inputPairs.Push(inputPairs)

	for ff.inputPairs.Used() > 0 {
		var inputPairBuf [1][2]Input
		ff.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]
		ff.tick = ip[0].Tick
		ff.core.SetKeys(mgba.Keys(ip[ff.localPlayerIndex].Joyflags))
		ff.advanceOne()
		if err := il.Write(ff.bn6.RNG2State(ff.core), ip); err != nil {
			return nil, nil, err
		}
	}

	committedState := ff.state
	if committedState == nil {
		return nil, nil, errors.New("no committed state?")
	}

	// Run the local inputs and predict what the remote side did and create the new dirty state.
	lastRemoteInput := inputPairs[len(inputPairs)-1][1-localPlayerIndex]

	predictedInputPairs := make([][2]Input, len(localPlayerInputsLeft))
	for i, inp := range localPlayerInputsLeft {
		predictedInputPairs[i][localPlayerIndex] = inp

		predicted := &predictedInputPairs[i][1-localPlayerIndex]
		predicted.Tick = inp.Tick
		ff.tick = predicted.Tick
		predicted.CustomScreenState = lastRemoteInput.CustomScreenState
		if lastRemoteInput.Joyflags&uint16(mgba.KeysA) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysA)
		}
		if lastRemoteInput.Joyflags&uint16(mgba.KeysB) != 0 {
			predicted.Joyflags |= uint16(mgba.KeysB)
		}
	}

	ff.inputPairs = ringbuf.New[[2]Input](len(localPlayerInputsLeft))
	ff.inputPairs.Push(predictedInputPairs)

	for ff.inputPairs.Used() > 0 {
		var inputPairBuf [1][2]Input
		ff.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]
		ff.core.SetKeys(mgba.Keys(ip[ff.localPlayerIndex].Joyflags))
		ff.advanceOne()
	}

	dirtyState := ff.state
	if dirtyState == nil {
		return nil, nil, errors.New("no committed state?")
	}

	return committedState, dirtyState, nil
}
