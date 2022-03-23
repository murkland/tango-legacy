package mgba

/*
#include <mgba/core/core.h>
#include <mgba/internal/gba/gba.h>

typedef void bbn6_mgba_bkpt16_irqh(struct ARMCore* cpu, int immediate);

bool bbn6_mgba_mCore_init(struct mCore* core) {
	return core->init(core);
}

void bbn6_mgba_mCore_deinit(struct mCore* core) {
	return core->deinit(core);
}

void bbn6_mgba_mCore_getGameTitle(struct mCore* core, char* title) {
	core->getGameTitle(core, title);
}

void bbn6_mgba_mCore_getGameCode(struct mCore* core, char* code) {
	core->getGameCode(core, code);
}

void bbn6_mgba_mCore_desiredVideoDimensions(const struct mCore* core, unsigned* width, unsigned* height) {
	core->desiredVideoDimensions(core, width, height);
}

void bbn6_mgba_mCore_setVideoBuffer(struct mCore* core, color_t* buffer, size_t stride) {
	core->setVideoBuffer(core, buffer, stride);
}

void bbn6_mgba_mCore_reset(struct mCore* core) {
	core->reset(core);
}

void bbn6_mgba_mCore_setSync(struct mCore* core, struct mCoreSync* sync) {
	core->setSync(core, sync);
}

void bbn6_mgba_mCore_runFrame(struct mCore* core) {
	core->runFrame(core);
}

int32_t bbn6_mgba_mCore_frequency(struct mCore* core) {
	return core->frequency(core);
}

uint32_t bbn6_mgba_mCore_frameCounter(struct mCore* core) {
	return core->frameCounter(core);
}

struct blip_t* bbn6_mgba_mCore_getAudioChannel(struct mCore* core, int ch) {
	return core->getAudioChannel(core, ch);
}
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

type CoreOptions struct {
	AudioBuffers int
	SampleRate   int
	AudioSync    bool
	VideoSync    bool
	Volume       int
}

type Core struct {
	ptr            *C.struct_mCore
	realBkpt16Irqh *C.bbn6_mgba_bkpt16_irqh
	beefTrap       func()
}

func FindCore(fn string) (*Core, error) {
	fnCstr := C.CString(fn)
	defer C.free(unsafe.Pointer(fnCstr))

	ptr := C.mCoreFind(fnCstr)
	if ptr == nil {
		return nil, fmt.Errorf("could not find core for %s", fn)
	}

	core := &Core{ptr, nil, nil}

	if !C.bbn6_mgba_mCore_init(core.ptr) {
		return nil, errors.New("could not initialize core")
	}

	runtime.SetFinalizer(core, func(core *Core) {
		core.Close()
	})

	return core, nil
}

func (c *Core) SetOptions(o CoreOptions) {
	c.ptr.opts.audioBuffers = C.ulong(o.AudioBuffers)
	c.ptr.opts.sampleRate = C.uint(o.SampleRate)
	c.ptr.opts.audioSync = C.bool(o.AudioSync)
	c.ptr.opts.videoSync = C.bool(o.VideoSync)
	c.ptr.opts.volume = C.int(o.Volume)
}

func (c *Core) Options() CoreOptions {
	return CoreOptions{
		AudioBuffers: int(c.ptr.opts.audioBuffers),
		SampleRate:   int(c.ptr.opts.sampleRate),
		AudioSync:    bool(c.ptr.opts.audioSync),
		VideoSync:    bool(c.ptr.opts.videoSync),
		Volume:       int(c.ptr.opts.volume),
	}
}

func (c *Core) DesiredVideoDimensions() (int, int) {
	var width C.uint
	var height C.uint
	C.bbn6_mgba_mCore_desiredVideoDimensions(c.ptr, &width, &height)
	return int(width), int(height)
}

func (c *Core) SetVideoBuffer(buf unsafe.Pointer, width int) {
	C.bbn6_mgba_mCore_setVideoBuffer(c.ptr, (*C.uint)(buf), C.ulong(width))
}

func (c *Core) LoadFile(path string) error {
	pathCstr := C.CString(path)
	defer C.free(unsafe.Pointer(pathCstr))
	if !C.mCoreLoadFile(c.ptr, pathCstr) {
		return fmt.Errorf("could not load file %s", path)
	}
	return nil
}

func (c *Core) GameTitle() string {
	var title [16]byte
	C.bbn6_mgba_mCore_getGameTitle(c.ptr, (*C.char)(unsafe.Pointer(&title)))
	return string(bytes.TrimRight(title[:], "\x00"))
}

func (c *Core) GameCode() string {
	var code [8]byte
	C.bbn6_mgba_mCore_getGameCode(c.ptr, (*C.char)(unsafe.Pointer(&code)))
	return string(code[:])
}

func (c *Core) AutoloadSave() bool {
	return bool(C.mCoreAutoloadSave(c.ptr))
}

func (c *Core) Config() *Config {
	return &Config{&c.ptr.config}
}

func (c *Core) LoadConfig() {
	C.mCoreLoadConfig(c.ptr)
}

func (c *Core) Reset() {
	C.bbn6_mgba_mCore_reset(c.ptr)
}

func (c *Core) RunFrame() {
	C.bbn6_mgba_mCore_runFrame(c.ptr)
}

func (c *Core) Frequency() int32 {
	return int32(C.bbn6_mgba_mCore_frequency(c.ptr))
}

func (c *Core) FrameCounter() uint32 {
	return uint32(C.bbn6_mgba_mCore_frameCounter(c.ptr))
}

func (c *Core) AudioChannel(ch int) *Blip {
	return &Blip{C.bbn6_mgba_mCore_getAudioChannel(c.ptr, C.int(ch))}
}

func (c *Core) SetSync(sync *Sync) {
	C.bbn6_mgba_mCore_setSync(c.ptr, sync.ptr)
}

func (c *Core) GBA() *GBA {
	if c.ptr.board == nil {
		return nil
	}
	return &GBA{(*C.struct_GBA)(c.ptr.board)}
}

func (c *Core) Close() {
	if c.ptr == nil {
		return
	}
	C.bbn6_mgba_mCore_deinit(c.ptr)
	c.Config().Deinit()
	c.ptr = nil
}
