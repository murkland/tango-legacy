package bn6

import (
	"log"

	"github.com/murkland/bbn6/mgba"
)

func push(core *mgba.Core, regs ...int) {
	sp := core.GBA().Register(13) - uint32(len(regs)*4)
	core.GBA().SetRegister(13, sp)

	address := sp
	for _, r := range regs {
		core.RawWrite32(address, -1, core.GBA().Register(r))
		address += 4
	}
}

func pop(core *mgba.Core, regs ...int) {
	address := core.GBA().Register(13)
	core.GBA().SetRegister(13, address+uint32(len(regs)*4))

	for _, r := range regs {
		core.GBA().SetRegister(r, core.RawRead32(address, -1))
		address += 4
	}
}

func MakeIRQFFTrap(core *mgba.Core, offsets Offsets) mgba.IRQTrap {
	return func() {
		caller := core.GBA().Register(15) - 4

		if caller == offsets.A_battle_init__call__battle_copyInputData {
			core.GBA().SetRegister(0, 0)

			// TODO: init code

			// Skip the call entirely.
			return
		}

		if caller == offsets.A_battle_update__call__battle_copyInputData {
			core.GBA().SetRegister(0, 0)

			// TODO: get inputs

			// Skip the call entirely.
			return
		}

		if caller == offsets.A_battle_init_marshal__ret {
			// TODO
			pop(core, 4, 6, 15)
			return
		}

		if caller == offsets.A_battle_turn_marshal__ret {
			// TODO
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
