package av

import (
	"runtime"
	"unsafe"
)

/*
#include <stdlib.h>
*/
import "C"

type VideoBuffer struct {
	buf    unsafe.Pointer
	width  int
	height int
}

func NewVideoBuffer(width int, height int) *VideoBuffer {
	buf := C.malloc(C.size_t(width * height * 4))
	vb := &VideoBuffer{buf, width, height}
	runtime.SetFinalizer(vb, func(vb *VideoBuffer) {
		C.free(vb.buf)
	})
	return vb
}

func (vb *VideoBuffer) Pointer() unsafe.Pointer {
	return vb.buf
}

func (vb *VideoBuffer) Pix() []byte {
	return unsafe.Slice((*byte)(vb.buf), vb.width*vb.height*4)
}
