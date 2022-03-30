package mgba

/*
#include <mgba-util/vfs.h>

bool bbn6_mgba_util_VFile_close(struct VFile* vf) {
	return vf->close(vf);
}
*/
import "C"
import (
	"os"
	"unsafe"
)

type VFile struct {
	ptr *C.struct_VFile
}

func OpenVF(path string, flags int) *VFile {
	realFlags := 0
	if flags&os.O_APPEND != 0 {
		realFlags |= C.O_APPEND
	}
	if flags&os.O_CREATE != 0 {
		realFlags |= C.O_CREAT
	}
	if flags&os.O_EXCL != 0 {
		realFlags |= C.O_EXCL
	}
	if flags&os.O_RDONLY != 0 {
		realFlags |= C.O_RDONLY
	}
	if flags&os.O_RDWR != 0 {
		realFlags |= C.O_RDWR
	}
	if flags&os.O_TRUNC != 0 {
		realFlags |= C.O_TRUNC
	}
	if flags&os.O_WRONLY != 0 {
		realFlags |= C.O_WRONLY
	}

	pathCstr := C.CString(path)
	defer C.free(unsafe.Pointer(pathCstr))
	ptr := C.VFileOpen(pathCstr, C.int(realFlags))
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
