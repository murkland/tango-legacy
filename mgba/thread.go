package mgba

/*
#include <mgba/core/core.h>
#include <mgba/core/thread.h>
*/
import "C"
import (
	"runtime"
	"unsafe"
)

type Thread struct {
	t *C.struct_mCoreThread
}

func NewThread(core *Core) *Thread {
	t := &Thread{}
	t.t = (*C.struct_mCoreThread)(C.calloc(1, C.size_t(unsafe.Sizeof(C.struct_mCoreThread{}))))
	t.t.core = core.ptr
	t.t.logger.d = *C.mLogGetContext()

	runtime.SetFinalizer(t, func(t *Thread) {
		C.free(unsafe.Pointer(t.t))
	})
	return t
}

func (t *Thread) Start() bool {
	return bool(C.mCoreThreadStart(t.t))
}

func (t *Thread) Pause() {
	C.mCoreThreadPause(t.t)
}

func (t *Thread) Unpause() {
	C.mCoreThreadUnpause(t.t)
}

func (t *Thread) HasStarted() bool {
	return bool(C.mCoreThreadHasStarted(t.t))
}

func (t *Thread) HasExited() bool {
	return bool(C.mCoreThreadHasExited(t.t))
}

func (t *Thread) HasCrashed() bool {
	return bool(C.mCoreThreadHasCrashed(t.t))
}

func (t *Thread) End() {
	C.mCoreThreadEnd(t.t)
}

func (t *Thread) Join() {
	C.mCoreThreadJoin(t.t)
}
