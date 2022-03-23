package mgba

import "unsafe"

/*
#include <mgba/internal/gba/gba.h>

typedef void bbn6_mgba_swi16_handler_cb(struct ARMCore* cpu, int immediate);

void bbn6_cgo_hijackedGBASWI16IRQH(struct ARMCore* cpu, int immediate);

void bbn6_cgo_hijackedGBASWI16IRQH_trampoline(struct ARMCore* cpu, int immediate) {
	bbn6_cgo_hijackedGBASWI16IRQH(cpu, immediate);
}

void bbn6_mgba_ARMInterruptHandler_setSwi16(struct ARMInterruptHandler* irqh, bbn6_mgba_swi16_handler_cb* cb) {
	irqh->swi16 = cb;
}

void bbn6_call_mgba_swi16_handler_cb(bbn6_mgba_swi16_handler_cb* cb, struct ARMCore* cpu, int immediate) {
	cb(cpu, immediate);
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

type IRQTrap func(imm int) bool

func (c *Core) InstallGBASWI16IRQHTrap(irqTrap IRQTrap) {
	gba := ((*C.struct_GBA)(c.ptr.board))
	if c.realSwi16irq == nil {
		armCoreToMGBACoreMapping.Set(unsafe.Pointer(gba.cpu), c)
		c.realSwi16irq = (*C.bbn6_mgba_swi16_handler_cb)(unsafe.Pointer(gba.cpu.irqh.swi16))
		C.bbn6_mgba_ARMInterruptHandler_setSwi16(&gba.cpu.irqh, (*C.bbn6_mgba_swi16_handler_cb)(C.bbn6_cgo_hijackedGBASWI16IRQH_trampoline))
	}
	c.swi16irqTrap = irqTrap
}
