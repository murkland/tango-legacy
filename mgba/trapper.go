package mgba

/*
#include <mgba/core/cpu.h>
#include <mgba/internal/gba/gba.h>

struct Trapper {
	struct mCPUComponent cpuComponent;
	void (*realBkpt16)(struct ARMCore* cpu, int immediate);
	void* userData;
};

extern void tango_Trapper_initCallback(void* cpu, struct mCPUComponent* component);
extern void tango_Trapper_deinitCallback(struct mCPUComponent* component);
extern void tango_Trapper_handle(struct Trapper* component);

static void tango_mCPUComponent_setCallbacks_Trapper(struct mCPUComponent* cpuComponent) {
	cpuComponent->init = &tango_Trapper_initCallback;
	cpuComponent->deinit = &tango_Trapper_deinitCallback;
}

static void tango_Trapper_bkpt16(struct ARMCore* cpu, int immediate) {
	struct GBA* gba = (struct GBA*) cpu->master;
	struct Trapper* component = (struct Trapper*) gba->cpu->components[CPU_COMPONENT_MISC_1];
	if (immediate == 0xef) {
		tango_Trapper_handle(component);
		return;
	}
	component->realBkpt16(cpu, immediate);
}

static void tango_mgba_ARMInterruptHandler_setBkpt16_Trapper(struct ARMInterruptHandler* irqh) {
	irqh->bkpt16 = tango_Trapper_bkpt16;
}
*/
import "C"
import (
	"fmt"
	"runtime"
	"runtime/cgo"
	"unsafe"
)

//export tango_Trapper_initCallback
func tango_Trapper_initCallback(cpu unsafe.Pointer, component *C.struct_mCPUComponent) {
	handle := *(*cgo.Handle)((*C.struct_Trapper)(unsafe.Pointer(component)).userData)
	t := handle.Value().(*Trapper)
	t.Init(cpu)
}

//export tango_Trapper_deinitCallback
func tango_Trapper_deinitCallback(component *C.struct_mCPUComponent) {
	handle := *(*cgo.Handle)((*C.struct_Trapper)(unsafe.Pointer(component)).userData)
	t := handle.Value().(*Trapper)
	t.Deinit()
}

//export tango_Trapper_handle
func tango_Trapper_handle(component *C.struct_Trapper) {
	handle := *(*cgo.Handle)(component.userData)
	t := handle.Value().(*Trapper)
	t.Handle()
}

const trapOpcode = 0xbeef // bkpt 0xef

type trap struct {
	original uint16
	handler  func()
}

type Trapper struct {
	core   *Core
	ptr    *C.struct_Trapper
	handle cgo.Handle
	traps  map[uint32]trap
}

func NewTrapper(core *Core) *Trapper {
	t := &Trapper{core: core, traps: map[uint32]trap{}}
	t.ptr = (*C.struct_Trapper)(C.calloc(1, C.size_t(unsafe.Sizeof(C.struct_Trapper{}))))
	t.handle = cgo.NewHandle(t)
	t.ptr.userData = unsafe.Pointer(&t.handle)
	C.tango_mCPUComponent_setCallbacks_Trapper(&t.ptr.cpuComponent)
	runtime.SetFinalizer(t, func(t *Trapper) {
		t.handle.Delete()
		C.free(unsafe.Pointer(t.ptr))
	})
	return t
}

func (t *Trapper) Add(addr uint32, handler func()) {
	if _, ok := t.traps[addr]; ok {
		panic(fmt.Sprintf("trap at 0x%08x already exists", addr))
	}
	tr := trap{t.core.RawRead16(addr, -1), handler}
	t.core.RawWrite16(addr, -1, trapOpcode)
	t.traps[addr] = tr
}

func (t *Trapper) Init(cpu unsafe.Pointer) {
}

func (t *Trapper) Handle() {
	const wordSizeThumb = 2
	caller := t.core.GBA().Register(15) - wordSizeThumb*2

	trap := t.traps[caller]
	if trap.handler == nil {
		panic(fmt.Sprintf("unhandled trap at 0x%08x", caller))
	}

	t.core.GBA().ARMRunFake(trap.original)
	trap.handler()
}

func (t *Trapper) Deinit() {
}

func (t *Trapper) Attach(g *GBA) {
	armCore := (*C.struct_ARMCore)(g.ptr.cpu)
	t.ptr.realBkpt16 = armCore.irqh.bkpt16
	(*[C.CPU_COMPONENT_MAX]*C.struct_mCPUComponent)(unsafe.Pointer(armCore.components))[C.CPU_COMPONENT_MISC_1] = (*C.struct_mCPUComponent)(unsafe.Pointer(t.ptr))
	C.ARMHotplugAttach(armCore, C.CPU_COMPONENT_MISC_1)
	C.tango_mgba_ARMInterruptHandler_setBkpt16_Trapper(&armCore.irqh)
}
