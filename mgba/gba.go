package mgba

/*
#include <mgba/internal/gba/gba.h>
#include <mgba/internal/arm/isa-inlines.h>

void bbn6_mgba_ARMMemory_setActiveRegion(struct ARMCore* cpu, uint32_t address) {
	cpu->memory.setActiveRegion(cpu, address);
}
*/
import "C"
import (
	"unsafe"
)

type GBA struct {
	ptr *C.struct_GBA
}

func (g *GBA) armRegisterFile() *C.struct_ARMRegisterFile {
	return (*C.struct_ARMRegisterFile)(unsafe.Pointer(&g.ptr.cpu.anon0))
}

func (g *GBA) Register(r int) uint32 {
	return uint32(g.armRegisterFile().anon0.gprs[r])
}

func (g *GBA) CPSR() [4]byte {
	return g.armRegisterFile().anon0.cpsr
}

func (g *GBA) SPSR() [4]byte {
	return g.armRegisterFile().anon0.spsr
}

func (g *GBA) SetRegister(r int, v uint32) {
	g.armRegisterFile().anon0.gprs[r] = C.int(v)
}

func (g *GBA) SetCPSR(cpsr [4]byte) {
	g.armRegisterFile().anon0.cpsr = cpsr
}

func (g *GBA) SetSPSR(spsr [4]byte) {
	g.armRegisterFile().anon0.spsr = spsr
}

func (g *GBA) Sync() *Sync {
	if g.ptr.sync == nil {
		return nil
	}
	return &Sync{g.ptr.sync}
}

func (g *GBA) ThumbWritePC() {
	g.ptr.cpu.cycles += C.int(1 + g.ptr.cpu.memory.activeSeqCycles16 + C.uint(C.ThumbWritePC(g.ptr.cpu)))
}

func GBAAudioCalculateRatio(inputSampleRate float32, desiredFPS float32, desiredSampleRate float32) float32 {
	return float32(C.GBAAudioCalculateRatio(C.float(inputSampleRate), C.float(desiredFPS), C.float(desiredSampleRate)))
}
