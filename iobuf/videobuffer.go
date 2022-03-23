package iobuf

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
	vb := &VideoBuffer{
		width:  width,
		height: height,
		buf:    unsafe.Pointer(C.calloc(1, C.ulong(width*height*4))),
	}
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
	for i := 0; i < vb.width*vb.height*4; i++ {
		pix := uint8(*(*C.uint8_t)(unsafe.Pointer(uintptr(vb.buf) + uintptr(i))))
		if i%4 == 3 {
			pix = 0xff
		}
		img.Pix[i] = pix
	}
	return img
}
