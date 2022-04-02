package mgba

import (
	"unsafe"
)

/*
#include <mgba/internal/gba/gba.h>

typedef void tango_mgba_bkpt16_irqh(struct ARMCore* cpu, int immediate);
void tango_call_mgba_bkpt16_irqh(tango_mgba_bkpt16_irqh* irqh, struct ARMCore* cpu, int immediate);
*/
import "C"

//export tango_cgo_hijackedGBABkpt16IRQH
func tango_cgo_hijackedGBABkpt16IRQH(armCore unsafe.Pointer, imm int) {
	c := armCoreToMGBACoreMapping.Get(armCore)
	if c.beefTrap != nil && imm == 0xef {
		c.beefTrap()
	}
	C.tango_call_mgba_bkpt16_irqh(c.realBkpt16Irqh, (*C.struct_ARMCore)(armCore), C.int(imm))
}
