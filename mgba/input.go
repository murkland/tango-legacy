package mgba

/*
#include <mgba/core/core.h>

void bbn6_mgba_mCore_setKeys(struct mCore* core, uint32_t keys) {
	core->setKeys(core, keys);
}
*/
import "C"

type Keys uint32

const (
	KeysA      Keys = 0b0000000001
	KeysB      Keys = 0b0000000010
	KeysSelect Keys = 0b0000000100
	KeysStart  Keys = 0b0000001000
	KeysRight  Keys = 0b0000010000
	KeysLeft   Keys = 0b0000100000
	KeysUp     Keys = 0b0001000000
	KeysDown   Keys = 0b0010000000
	KeysR      Keys = 0b0100000000
	KeysL      Keys = 0b1000000000
)

func (c *Core) SetKeys(keys Keys) {
	C.bbn6_mgba_mCore_setKeys(c.ptr, C.uint32_t(keys))
}
