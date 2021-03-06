package mgba

/*
#include <mgba/core/core.h>

uint32_t tango_mgba_mCore_rawRead8(struct mCore* core, uint32_t address, int segment) {
	return core->rawRead8(core, address, segment);
}
uint32_t tango_mgba_mCore_rawRead16(struct mCore* core, uint32_t address, int segment) {
	return core->rawRead16(core, address, segment);
}
uint32_t tango_mgba_mCore_rawRead32(struct mCore* core, uint32_t address, int segment) {
	return core->rawRead32(core, address, segment);
}

void tango_mgba_mCore_rawWrite8(struct mCore* core, uint32_t address, int segment, uint8_t v) {
	return core->rawWrite8(core, address, segment, v);
}
void tango_mgba_mCore_rawWrite16(struct mCore* core, uint32_t address, int segment, uint16_t v) {
	return core->rawWrite16(core, address, segment, v);
}
void tango_mgba_mCore_rawWrite32(struct mCore* core, uint32_t address, int segment, uint32_t v) {
	return core->rawWrite32(core, address, segment, v);
}
*/
import "C"

func (c *Core) RawRead8(address uint32, segment int) uint8 {
	return uint8(C.tango_mgba_mCore_rawRead8(c.ptr, C.uint32_t(address), C.int(segment)))
}

func (c *Core) RawRead16(address uint32, segment int) uint16 {
	return uint16(C.tango_mgba_mCore_rawRead16(c.ptr, C.uint32_t(address), C.int(segment)))
}

func (c *Core) RawRead32(address uint32, segment int) uint32 {
	return uint32(C.tango_mgba_mCore_rawRead32(c.ptr, C.uint32_t(address), C.int(segment)))
}

func (c *Core) RawReadRange(address uint32, segment int, buf []byte) {
	for i := range buf {
		buf[i] = c.RawRead8(address+uint32(i), segment)
	}
}

func (c *Core) RawWrite8(address uint32, segment int, v uint8) {
	C.tango_mgba_mCore_rawWrite8(c.ptr, C.uint32_t(address), C.int(segment), C.uint8_t(v))
}

func (c *Core) RawWrite16(address uint32, segment int, v uint16) {
	C.tango_mgba_mCore_rawWrite16(c.ptr, C.uint32_t(address), C.int(segment), C.uint16_t(v))
}

func (c *Core) RawWrite32(address uint32, segment int, v uint32) {
	C.tango_mgba_mCore_rawWrite32(c.ptr, C.uint32_t(address), C.int(segment), C.uint32_t(v))
}

func (c *Core) RawWriteRange(address uint32, segment int, buf []byte) {
	for i, v := range buf {
		c.RawWrite8(address+uint32(i), segment, v)
	}
}
