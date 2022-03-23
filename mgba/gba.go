package mgba

/*
#include <mgba/internal/gba/gba.h>
*/
import "C"
import "unsafe"

type GBA struct {
	ptr *C.struct_GBA
}

func (g *GBA) armRegisterFile() *C.struct_ARMRegisterFile {
	return (*C.struct_ARMRegisterFile)(unsafe.Pointer(&g.ptr.cpu.anon0))
}

func (g *GBA) Register(r int) uint32 {
	return uint32(g.armRegisterFile().anon0.gprs[r])
}

func (g *GBA) SetRegister(r int, v uint32) {
	g.armRegisterFile().anon0.gprs[r] = C.int(v)
}

func (g *GBA) Sync() *Sync {
	if g.ptr.sync == nil {
		return nil
	}
	return &Sync{g.ptr.sync}
}

func GBAAudioCalculateRatio(inputSampleRate float32, desiredFPS float32, desiredSampleRate float32) float32 {
	return float32(C.GBAAudioCalculateRatio(C.float(inputSampleRate), C.float(desiredFPS), C.float(desiredSampleRate)))
}
