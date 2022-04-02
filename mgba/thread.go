package mgba

/*
#include <mgba/core/core.h>
#include <mgba/core/thread.h>

extern void tango_mCoreThread_frameCallback(struct mCoreThread* threadContext);
*/
import "C"
import (
	"runtime"
	"runtime/cgo"
	"unsafe"
)

type Thread struct {
	ptr    *C.struct_mCoreThread
	handle cgo.Handle

	frameCallback func()
}

//export tango_mCoreThread_frameCallback
func tango_mCoreThread_frameCallback(ptr *C.struct_mCoreThread) {
	handle := *(*cgo.Handle)(ptr.userData)
	t := handle.Value().(*Thread)
	t.frameCallback()
}

func NewThread(core *Core) *Thread {
	t := &Thread{}
	t.ptr = (*C.struct_mCoreThread)(C.calloc(1, C.size_t(unsafe.Sizeof(C.struct_mCoreThread{}))))
	t.ptr.core = core.ptr
	t.ptr.logger.d = *C.mLogGetContext()
	t.handle = cgo.NewHandle(t)
	t.ptr.userData = unsafe.Pointer(&t.handle)
	t.ptr.frameCallback = C.ThreadCallback(C.tango_mCoreThread_frameCallback)

	runtime.SetFinalizer(t, func(t *Thread) {
		t.handle.Delete()
		C.free(unsafe.Pointer(t.ptr))
	})
	return t
}

func (t *Thread) Start() bool {
	return bool(C.mCoreThreadStart(t.ptr))
}

func (t *Thread) Pause() {
	C.mCoreThreadPause(t.ptr)
}

func (t *Thread) Unpause() {
	C.mCoreThreadUnpause(t.ptr)
}

func (t *Thread) IsPaused() bool {
	return bool(C.mCoreThreadIsPaused(t.ptr))
}

func (t *Thread) HasStarted() bool {
	return bool(C.mCoreThreadHasStarted(t.ptr))
}

func (t *Thread) HasExited() bool {
	return bool(C.mCoreThreadHasExited(t.ptr))
}

func (t *Thread) HasCrashed() bool {
	return bool(C.mCoreThreadHasCrashed(t.ptr))
}

func (t *Thread) End() {
	C.mCoreThreadEnd(t.ptr)
}

func (t *Thread) Join() {
	C.mCoreThreadJoin(t.ptr)
}

func (t *Thread) SetFrameCallback(f func()) {
	t.frameCallback = f
}
