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
		if caller == offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput {
			StartBattleFromCommMenu(core)
		} else if caller == offsets.A_commMenu_handleLinkCableInput__entry {
			log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", caller)
			push(core, 4, 5, 6, 7, 14)
		}
	}
}
