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
	commitTime       int
	committedState   *mgba.State
	dirtyState       *mgba.State
	rw               *replay.Writer
}

func NewFastforwarder(romPath string, bn6 *bn6.BN6) (*Fastforwarder, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	ff := &Fastforwarder{core, bn6, nil, 0}

	tp := mgba.NewTrapper(core)

	tp.Add(bn6.Offsets.ROM.A_main__readJoyflags, func() {
		if int(ff.bn6.InBattleTime(ff.core)) == ff.state.commitTime {
			ff.state.committedState = core.SaveState()
		}

		// The dirty state is 1 tick before all input pairs, such that the final input can run on the main core safely.
		if ff.state.inputPairs.Used() == 1 {
			ff.state.dirtyState = core.SaveState()
		}

		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		if ip[0].LocalTick != ip[1].LocalTick {
			ff.state.err = fmt.Errorf("p1 tick != p2 tick: %d != %d", ip[0].LocalTick, ip[1].LocalTick)
			return
		}

		inBattleTime := int(ff.bn6.InBattleTime(ff.core))
		if ip[0].LocalTick != inBattleTime {
			ff.state.err = fmt.Errorf("input tick != in battle tick: %d != %d", ip[0].LocalTick, inBattleTime)
			return
		}

		core.GBA().SetRegister(4, uint32(ip[ff.state.localPlayerIndex].Joyflags))
	})

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		core.GBA().SetRegister(0, 0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+4)
		core.GBA().ThumbWritePC()

		if ff.state.inputPairs.Used() == 0 {
			return
		}

		var inputPairBuf [1][2]input.Input
		ff.state.inputPairs.Pop(inputPairBuf[:], 0)
		ip := inputPairBuf[0]

		bn6.SetPlayerInputState(core, 0, ip[0].Joyflags, ip[0].CustomScreenState)
		if ip[0].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 0, ip[0].Turn)
			log.Printf("p1 turn committed at tick %d", ip[0].LocalTick)
		}

		bn6.SetPlayerInputState(core, 1, ip[1].Joyflags, ip[1].CustomScreenState)
		if ip[1].Turn != nil {
			bn6.SetPlayerMarshaledBattleState(core, 1, ip[1].Turn)
			log.Printf("p2 turn committed at tick %d", ip[1].LocalTick)
		}

		inBattleTime := int(ff.bn6.InBattleTime(ff.core))
		if inBattleTime < ff.state.commitTime {
			if err := ff.state.rw.Write(ff.bn6.RNG2State(ff.core), ip); err != nil {
				ff.state.err = err
				return
			}
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

// Fastforward fastfowards the state to the new state.
//
// The committed state MAY be after the dirty state -- the dirty state is exactly 1 tick before the final state, and the caller must make sure to run the inputs in its own core, if they exist.
//
// BEWARE: only one thread may call fastforward at a time.
func (ff *Fastforwarder) Fastforward(state *mgba.State, rw *replay.Writer, localPlayerIndex int, inputPairs [][2]input.Input, lastCommittedRemoteInput input.Input, localPlayerInputsLeft []input.Input) (*mgba.State, *mgba.State, *[2]input.Input, error) {
	startTime := time.Now()

	if !ff.core.LoadState(state) {
		return nil, nil, nil, errors.New("failed to load state")
	}

	startInBattleTime := int(ff.bn6.InBattleTime(ff.core))
	commitTime := startInBattleTime + len(inputPairs)

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

	inputPairs = append(inputPairs, predictedInputPairs...)

	// Rewind state to a point where inputs can be applied safely.
	ff.core.GBA().SetRegister(15, ff.bn6.Offsets.ROM.A_main__readJoyflags)
	ff.core.GBA().ThumbWritePC()

	lastInput := inputPairs[len(inputPairs)-1]

	ff.state = &fastforwarderState{
		localPlayerIndex: localPlayerIndex,
		inputPairs:       ringbuf.New[[2]input.Input](len(inputPairs)),
		commitTime:       commitTime,
		rw:               rw,
	}
	defer func() {
		ff.state = nil
	}()
	ff.state.inputPairs.Push(inputPairs)

	for ff.state.inputPairs.Used() > 0 {
		ff.state.err = nil
		ff.core.RunFrame()
		if ff.state.err != nil {
			return nil, nil, nil, ff.state.err
		}
	}

	ff.lastFastforwardDuration = time.Now().Sub(startTime)

	return ff.state.committedState, ff.state.dirtyState, &lastInput, nil
}
