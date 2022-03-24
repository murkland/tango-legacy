package av

import (
	"image"
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

func (vb *VideoBuffer) CopyImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, vb.width, vb.height))
	img.Pix = C.GoBytes(vb.buf, C.int(vb.width*vb.height*4))
	return img
}
