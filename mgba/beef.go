package mgba

import (
	"unsafe"
)

/*
#include <mgba/internal/gba/gba.h>

typedef void bbn6_mgba_bkpt16_irqh(struct ARMCore* cpu, int immediate);

void bbn6_cgo_hijackedGBABkpt16IRQH(struct ARMCore* cpu, int immediate);

void bbn6_cgo_hijackedGBABkpt16IRQH_trampoline(struct ARMCore* cpu, int immediate) {
	bbn6_cgo_hijackedGBABkpt16IRQH(cpu, immediate);
}

void bbn6_mgba_ARMInterruptHandler_setBkpt16(struct ARMInterruptHandler* irqh, bbn6_mgba_bkpt16_irqh* cb) {
	irqh->bkpt16 = cb;
}

void bbn6_call_mgba_bkpt16_irqh(bbn6_mgba_bkpt16_irqh* irqh, struct ARMCore* cpu, int immediate) {
	irqh(cpu, immediate);
}
*/
import "C"

type armToMGBACoreMap struct {
	m map[unsafe.Pointer]*Core
}

func newArmCoreToMGBACoreMapping() armToMGBACoreMap {
	return armToMGBACoreMap{map[unsafe.Pointer]*Core{}}
}

func (m *armToMGBACoreMap) Set(armCore unsafe.Pointer, c *Core) {
	m.m[armCore] = c
}

func (m *armToMGBACoreMap) Get(armCore unsafe.Pointer) *Core {
	return m.m[armCore]
}

var armCoreToMGBACoreMapping = newArmCoreToMGBACoreMapping()

func (c *Core) InstallBeefTrap(handler func()) {
	gba := c.GBA().ptr
	if c.realBkpt16Irqh == nil {
		armCoreToMGBACoreMapping.Set(unsafe.Pointer(gba.cpu), c)
		c.realBkpt16Irqh = (*C.bbn6_mgba_bkpt16_irqh)(unsafe.Pointer(gba.cpu.irqh.bkpt16))
		C.bbn6_mgba_ARMInterruptHandler_setBkpt16(&gba.cpu.irqh, (*C.bbn6_mgba_bkpt16_irqh)(C.bbn6_cgo_hijackedGBABkpt16IRQH_trampoline))
	}
	c.beefTrap = handler
}
