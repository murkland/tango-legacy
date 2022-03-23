package hijack

import "github.com/murkland/bbn6/mgba"

func PatchWithSVC(c *mgba.Core, addr uint32, imm uint8) {
	c.RawWrite8(addr, -1, imm)
	c.RawWrite8(addr+1, -1, 0xdf)
}

func PatchWithNOP(c *mgba.Core, addr uint32) {
	c.RawWrite8(addr, -1, 0xc0)
	c.RawWrite8(addr+1, -1, 0x46)
}
