package mgba

/*
#include <mgba-util/vfs.h>

bool bbn6_mgba_util_VFile_close(struct VFile* vf) {
	return vf->close(vf);
}
*/
import "C"
import (
	"unsafe"
)

type VFile struct {
	ptr *C.struct_VFile
}

func OpenVF(path string, flags int) *VFile {
	pathCstr := C.CString(path)
	defer C.free(unsafe.Pointer(pathCstr))
	ptr := C.VFileOpen(pathCstr, C.int(flags))
	if ptr == nil {
		return nil
	}
	vf := &VFile{ptr}
	return vf
}

func (vf *VFile) Close() bool {
	if vf.ptr == nil {
		return true
	}
	r := bool(C.bbn6_mgba_util_VFile_close(vf.ptr))
	if !r {
		return false
	}
	vf.ptr = nil
	return true
}
