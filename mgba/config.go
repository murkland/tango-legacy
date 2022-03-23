package mgba

/*
#include <mgba/core/core.h>
*/
import "C"
import "unsafe"

type Config struct {
	ptr *C.struct_mCoreConfig
}

func (c *Config) Load() {
	C.mCoreConfigLoad(c.ptr)
}

func (c *Config) Init(name string) {
	nameCstr := C.CString(name)
	defer C.free(unsafe.Pointer(nameCstr))
	C.mCoreConfigInit(c.ptr, nameCstr)
}

func (c *Config) Deinit() {
	C.mCoreConfigLoad(c.ptr)
}
