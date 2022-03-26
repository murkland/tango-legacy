package game

import (
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

func deserializeReplay(r io.Reader) (*Replay, error) {
	// TODO: Actually deserialize this.
	return nil, nil
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
