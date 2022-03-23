package mgba

/*
#include <mgba/core/sync.h>
*/
import "C"

type Sync struct {
	ptr *C.struct_mCoreSync
}

func (s *Sync) FPSTarget() float32 {
	return float32(s.ptr.fpsTarget)
}

func (s *Sync) LockAudio() {
	C.mCoreSyncLockAudio(s.ptr)
}

func (s *Sync) ConsumeAudio() {
	C.mCoreSyncConsumeAudio(s.ptr)
}

func (s *Sync) WaitFrameStart() bool {
	return bool(C.mCoreSyncWaitFrameStart(s.ptr))
}

func (s *Sync) WaitFrameEnd() {
	C.mCoreSyncWaitFrameEnd(s.ptr)
}
