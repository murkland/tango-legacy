package bn6

import (
	"log"

	"github.com/murkland/bbn6/mgba"
)

func MakeIRQFFTrap(core *mgba.Core, offsets Offsets) mgba.IRQTrap {
	return func() {
		caller := core.GBA().Register(15) - 4
		if caller == offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput {
			StartBattleFromCommMenu(core)
		} else if caller == offsets.A_commMenu_handleLinkCableInput__entry {
			log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", caller)

			// stmdb !sp, {r4, r5, r6, r7, lr}
			sp := core.GBA().Register(13)

			sp -= 4
			core.RawWrite32(sp, -1, core.GBA().Register(14))

			sp -= 4
			core.RawWrite32(sp, -1, core.GBA().Register(7))

			sp -= 4
			core.RawWrite32(sp, -1, core.GBA().Register(6))

			sp -= 4
			core.RawWrite32(sp, -1, core.GBA().Register(5))

			sp -= 4
			core.RawWrite32(sp, -1, core.GBA().Register(4))

			core.GBA().SetRegister(13, sp)
		}
	}
}
