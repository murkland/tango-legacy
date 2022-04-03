package game

import (
	"fmt"

	"github.com/murkland/ringbuf"
	"github.com/murkland/tango/bn6"
	"github.com/murkland/tango/input"
	"github.com/murkland/tango/mgba"
	"github.com/murkland/tango/replay"
)

type Replayer struct {
	core *mgba.Core
	bn6  *bn6.BN6

	replay *replay.Replay

	currentInputPairs *ringbuf.RingBuf[[2]input.Input]
}

func (rp *Replayer) Reset() {
	rp.currentInputPairs = ringbuf.New[[2]input.Input](len(rp.replay.InputPairs))
	rp.currentInputPairs.Push(rp.replay.InputPairs)
	rp.core.LoadState(rp.replay.State)
	rp.bn6.SetPlayerMarshaledBattleState(rp.core, 0, rp.replay.Init[0])
	rp.bn6.SetPlayerMarshaledBattleState(rp.core, 1, rp.replay.Init[1])
}

func (rp *Replayer) Core() *mgba.Core {
	return rp.core
}

func NewReplayer(romPath string, r *replay.Replay) (*Replayer, error) {
	core, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	bn6 := bn6.Load(core.GameTitle())
	if bn6 == nil {
		return nil, fmt.Errorf("unsupported game: %s", core.GameTitle())
	}

	rp := &Replayer{core, bn6, r, nil}

	tp := mgba.NewTrapper(core)

	tp.Add(bn6.Offsets.ROM.A_main__readJoyflags, func() {
		var inputPairBuf [1][2]input.Input
		rp.currentInputPairs.Peek(inputPairBuf[:], 0)
		ip := inputPairBuf[0]
		core.GBA().SetRegister(4, uint32(ip[rp.replay.LocalPlayerIndex].Joyflags))
	})

	tp.Add(bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		rp.core.GBA().SetRegister(0, 0)
		rp.core.GBA().SetRegister(15, rp.core.GBA().Register(15)+4)
		rp.core.GBA().ThumbWritePC()

		var inputPairBuf [1][2]input.Input
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

	tp.Add(bn6.Offsets.ROM.A_getCopyDataInputState__ret, func() {
		core.GBA().SetRegister(0, 2)
	})

	tp.Add(bn6.Offsets.ROM.A_commMenu_endBattle__entry, func() {
		rp.Reset()
	})

	tp.Attach(core.GBA())

	return rp, nil
}
