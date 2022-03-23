package bn6

import (
	"log"

	"github.com/murkland/bbn6/mgba"
)

func push(core *mgba.Core, regs ...int) {
	sp := core.GBA().Register(13) - uint32(len(regs)*4)
	core.GBA().SetRegister(13, sp)

	for _, r := range regs {
		core.RawWrite32(sp, -1, core.GBA().Register(r))
		sp += 4
	}
}

func pop(core *mgba.Core, regs ...int) {
	sp := core.GBA().Register(13)

	for _, r := range regs {
		core.GBA().SetRegister(r, core.RawRead32(sp, -1))
		sp += 4
		if r == 15 {
			core.GBA().ThumbWritePC()
		}
	}

	core.GBA().SetRegister(13, sp)
}

func blAbsolute(core *mgba.Core, addr uint32) {
	core.GBA().SetRegister(14, core.GBA().Register(15)&^uint32(1))
	core.GBA().SetRegister(15, addr)
	core.GBA().ThumbWritePC()
}

func MakeIRQFFTrap(core *mgba.Core, offsets Offsets) mgba.IRQTrap {
	return func() {
		caller := core.GBA().Register(15) - 4

		if caller == offsets.A_battle_init__call__battle_copyInputData {
			// TODO: Set this correctly.
			inLinkBattle := false

			if inLinkBattle {
				// TODO: Get inputs.
				core.GBA().SetRegister(0, 0)
			} else {
				blAbsolute(core, offsets.A_battle_copyInputData__entry)
			}
			return
		}

		if caller == offsets.A_battle_update__call__battle_copyInputData {
			// TODO: Set this correctly.
			inLinkBattle := false

			if inLinkBattle {
				// TODO: Get inputs.
				core.GBA().SetRegister(0, 0)
			} else {
				blAbsolute(core, offsets.A_battle_copyInputData__entry)
			}
			return
		}

		if caller == offsets.A_battle_init_marshal__ret {
			// TODO
			init := LocalMarshaledBattleState(core)
			log.Printf("battle init: %v", init)
			pop(core, 4, 6, 15)
			return
		}

		if caller == offsets.A_battle_turn_marshal__ret {
			// TODO
			turn := LocalMarshaledBattleState(core)
			log.Printf("battle turn: %v", turn)
			pop(core, 15)
			return
		}

		if caller == offsets.A_battle_updating__ret__go_to_custom_screen {
			// TODO
			pop(core, 4, 15)
			return
		}

		if caller == offsets.A_battle_start__ret {
			// TODO
			pop(core, 4, 5, 7, 15)
			return
		}

		if caller == offsets.A_battle_end__entry {
			// TODO
			push(core, 4, 6, 7, 14)
			return
		}

		if caller == offsets.A_battle_isRemote__tst {
			// TODO: Set isRemote
			isRemote := true
			cpsr := core.GBA().CPSR()
			apsr := &cpsr[3]

			if isRemote {
				core.GBA().SetRegister(0, 1)
				*apsr = *apsr | uint8(0b0100)
			} else {
				core.GBA().SetRegister(0, 0)
				*apsr = *apsr & ^uint8(0b0100)
			}

			core.GBA().SetCPSR(cpsr)
			return
		}

		if caller == offsets.A_link_isRemote__ret {
			// TODO: Set isRemote
			isRemote := true
			if isRemote {
				core.GBA().SetRegister(0, 1)
			} else {
				core.GBA().SetRegister(0, 0)
			}

			pop(core, 7, 15)
			return
		}

		if caller == offsets.A_commMenu_handleLinkCableInput__entry {
			log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", caller)
			push(core, 4, 5, 6, 7, 14)
			return
		}

		if caller == offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput {
			StartBattleFromCommMenu(core)
			// Skip the call entirely.
			return
		}

		if caller == offsets.A_commMenu_inBattle__call__commMenu_handleLinkCableInput {
			// Skip the call entirely.
			return
		}

		log.Fatalf("unhandled irq 0xff trap at 0x%08x!", caller)
	}
}
