package mgba

/*
#include <mgba/core/core.h>
#include <mgba/internal/gba/serialize.h>

size_t tango_mgba_mCore_stateSize(struct mCore* core) {
	return core->stateSize(core);
}

bool tango_mgba_mCore_saveState(struct mCore* core, void* state) {
	return core->saveState(core, state);
}

bool tango_mgba_mCore_loadState(struct mCore* core, void* state) {
	return core->loadState(core, state);
}
*/
import "C"
import (
	"bytes"
	"runtime"
	"unsafe"
)

type State struct {
	ROMTitle string
	ROMCRC32 uint32

	ptr  unsafe.Pointer
	size int
}

func toState(buf unsafe.Pointer, size int) *State {
	serialized := (*C.struct_GBASerializedState)(buf)
	s := &State{
		ROMTitle: string(bytes.TrimRight(C.GoBytes(unsafe.Pointer(&serialized.title[0]), 12), "\x00")),
		ROMCRC32: uint32(serialized.romCrc32),

		ptr:  buf,
		size: size,
	}
	runtime.SetFinalizer(s, func(s *State) {
		C.free(s.ptr)
	})
	return s
}

func (c *Core) SaveState() *State {
	size := int(C.tango_mgba_mCore_stateSize(c.ptr))
	buf := unsafe.Pointer(C.malloc(C.size_t(size)))
	ok := C.tango_mgba_mCore_saveState(c.ptr, buf)
	if !ok {
		C.free(buf)
		return nil
	}
	return toState(buf, size)
}

func (c *Core) LoadState(state *State) bool {
	return bool(C.tango_mgba_mCore_loadState(c.ptr, state.ptr))
}

func (s *State) Bytes() []byte {
	return C.GoBytes(s.ptr, C.int(s.size))
}

func StateFromBytes(b []byte) *State {
	return toState(C.CBytes(b), len(b))
}
