package mgba

import (
	"unsafe"
)

/*
#include <mgba/internal/gba/gba.h>

typedef void bbn6_mgba_bkpt16_irqh(struct ARMCore* cpu, int immediate);
void bbn6_call_mgba_bkpt16_irqh(bbn6_mgba_bkpt16_irqh* irqh, struct ARMCore* cpu, int immediate);
*/
import "C"

//export bbn6_cgo_hijackedGBABkpt16IRQH
func bbn6_cgo_hijackedGBABkpt16IRQH(armCore unsafe.Pointer, imm int) {
	c := armCoreToMGBACoreMapping.Get(armCore)
	if c.beefTrap != nil && imm == 0xef {
		c.beefTrap()
	}
	C.bbn6_call_mgba_bkpt16_irqh(c.realBkpt16Irqh, (*C.struct_ARMCore)(armCore), C.int(imm))
}
