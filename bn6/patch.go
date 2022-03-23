package bn6

import (
	"github.com/murkland/bbn6/asm"
	"github.com/murkland/bbn6/mgba"
)

func PatchWithSVCFFs(core *mgba.Core, offsets Offsets) {
	core.RawWriteRange(offsets.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, -1, asm.Flatten(
		asm.SVC(0xff),
		asm.NOP(),
	))

	core.RawWriteRange(offsets.A_commMenu_handleLinkCableInput__entry, -1, asm.Flatten(
		asm.SVC(0xff),
	))
}
