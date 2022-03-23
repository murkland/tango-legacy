package mgba

import "unsafe"

/*
#include <mgba/internal/gba/gba.h>

typedef void bbn6_mgba_swi16_handler_cb(struct ARMCore* cpu, int immediate);
void bbn6_call_mgba_swi16_handler_cb(bbn6_mgba_swi16_handler_cb* cb, struct ARMCore* cpu, int immediate);
*/
import "C"

//export bbn6_cgo_hijackedGBASWI16IRQH
func bbn6_cgo_hijackedGBASWI16IRQH(armCore unsafe.Pointer, imm int) {
	c := armCoreToMGBACoreMapping.Get(armCore)
	irqTrap := c.swi16irqTraps[imm]
	if irqTrap == nil {
		C.bbn6_call_mgba_swi16_handler_cb(c.realSwi16irq, (*C.struct_ARMCore)(armCore), C.int(imm))
		return
	}
	irqTrap()
}
