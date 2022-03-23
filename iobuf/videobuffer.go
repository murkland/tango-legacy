package iobuf

import (
	"image"
	"reflect"
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
	buf := C.malloc(C.ulong(width * height * 4))
	vb := &VideoBuffer{buf, width, height}
	runtime.SetFinalizer(vb, func(vb *VideoBuffer) {
		C.free(vb.buf)
	})
	return vb
}

func (vb *VideoBuffer) Pointer() unsafe.Pointer {
	return vb.buf
}

func (vb *VideoBuffer) Image() *image.RGBA {
	var pix []uint8
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&pix)))
	sliceHeader.Len = vb.width * vb.height * 4
	sliceHeader.Cap = sliceHeader.Len
	sliceHeader.Data = uintptr(vb.buf)
	return &image.RGBA{Rect: image.Rect(0, 0, vb.width, vb.height), Pix: pix, Stride: vb.width * 4}
}
