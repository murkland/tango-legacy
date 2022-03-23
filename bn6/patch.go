package bn6

import (
	"github.com/murkland/bbn6/asm"
	"github.com/murkland/bbn6/mgba"
)

func PatchWithSVCFFs(core *mgba.Core, offsets Offsets) {
	// bl battleCopyInputData
	core.RawWriteRange(offsets.A_battle_init__call__battle_copyInputData, -1, asm.Flatten(
		asm.SVC(0xff),
		asm.NOP(),
	))

	// bl battleCopyInputData
	core.RawWriteRange(offsets.A_battle_update__call__battle_copyInputData, -1, asm.Flatten(
		asm.SVC(0xff),
		asm.NOP(),
	))

	// pop {r4, r6, pc}
	core.RawWriteRange(offsets.A_battle_init_marshal__ret, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// pop {pc}
	core.RawWriteRange(offsets.A_battle_turn_marshal__ret, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// pop {r4, pc}
	core.RawWriteRange(offsets.A_battle_updating__ret__go_to_custom_screen, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// pop {r4, r5, r7, pc}
	core.RawWriteRange(offsets.A_battle_start__ret, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// push {r4, r6, r7, lr}
	core.RawWriteRange(offsets.A_battle_end__entry, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// tst r0, r0
	core.RawWriteRange(offsets.A_battle_isRemote__tst, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// pop {r7, pc}
	core.RawWriteRange(offsets.A_link_isRemote__ret, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// push {r4, r5, r6, r7, lr}
	core.RawWriteRange(offsets.A_commMenu_handleLinkCableInput__entry, -1, asm.Flatten(
		asm.SVC(0xff),
	))

	// bl commMenu_handleLinkCableInput
	core.RawWriteRange(offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, -1, asm.Flatten(
		asm.SVC(0xff),
		asm.NOP(),
	))

	// bl commMenu_handleLinkCableInput
	core.RawWriteRange(offsets.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, -1, asm.Flatten(
		asm.SVC(0xff),
		asm.NOP(),
	))
}
