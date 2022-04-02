package mgba

/*
#include <mgba/core/core.h>
#include <mgba/gba/core.h>
#include <mgba/internal/gba/gba.h>

typedef void tango_mgba_bkpt16_irqh(struct ARMCore* cpu, int immediate);

bool tango_mgba_mCore_init(struct mCore* core) {
	return core->init(core);
}

bool tango_mgba_mCore_loadROM(struct mCore* core, struct VFile* vf) {
	return core->loadROM(core, vf);
}

bool tango_mgba_mCore_loadSave(struct mCore* core, struct VFile* vf) {
	return core->loadSave(core, vf);
}

void tango_mgba_mCore_deinit(struct mCore* core) {
	core->deinit(core);
}

void tango_mgba_mCore_getGameTitle(struct mCore* core, char* title) {
	core->getGameTitle(core, title);
}

void tango_mgba_mCore_getGameCode(struct mCore* core, char* code) {
	core->getGameCode(core, code);
}

void tango_mgba_mCore_desiredVideoDimensions(const struct mCore* core, unsigned* width, unsigned* height) {
	core->desiredVideoDimensions(core, width, height);
}

void tango_mgba_mCore_setVideoBuffer(struct mCore* core, color_t* buffer, size_t stride) {
	core->setVideoBuffer(core, buffer, stride);
}

void tango_mgba_mCore_reset(struct mCore* core) {
	core->reset(core);
}

void tango_mgba_mCore_setSync(struct mCore* core, struct mCoreSync* sync) {
	core->setSync(core, sync);
}

void tango_mgba_mCore_runFrame(struct mCore* core) {
	core->runFrame(core);
}

int32_t tango_mgba_mCore_frequency(struct mCore* core) {
	return core->frequency(core);
}

uint32_t tango_mgba_mCore_frameCounter(struct mCore* core) {
	return core->frameCounter(core);
}

void tango_mgba_mCore_checksum(struct mCore* core, void* data, enum mCoreChecksumType type) {
	core->checksum(core, data, type);
}

struct blip_t* tango_mgba_mCore_getAudioChannel(struct mCore* core, int ch) {
	return core->getAudioChannel(core, ch);
}

size_t tango_mgba_mCore_getAudioBufferSize(struct mCore* core) {
	return core->getAudioBufferSize(core);
}

void tango_mgba_mCore_setAudioBufferSize(struct mCore* core, size_t samples) {
	core->setAudioBufferSize(core, samples);
}
*/
import "C"
import (
	"bytes"
	"errors"
	"runtime"
	"unsafe"
)

type CoreOptions struct {
	AudioBuffers int
	SampleRate   int
	AudioSync    bool
	VideoSync    bool
	Volume       int
	FPSTarget    float32
}

type Core struct {
	ptr            *C.struct_mCore
	config         *Config
	realBkpt16Irqh *C.tango_mgba_bkpt16_irqh
	beefTrap       func()
}

func NewGBACore() (*Core, error) {
	ptr := C.GBACoreCreate()
	if ptr == nil {
		return nil, errors.New("could not create core")
	}

	core := &Core{ptr, &Config{&ptr.config, false}, nil, nil}

	if !C.tango_mgba_mCore_init(core.ptr) {
		return nil, errors.New("could not initialize core")
	}

	runtime.SetFinalizer(core, func(core *Core) {
		core.Close()
	})

	return core, nil
}

func (c *Core) SetOptions(o CoreOptions) {
	c.ptr.opts.audioBuffers = C.size_t(o.AudioBuffers)
	c.ptr.opts.sampleRate = C.uint(o.SampleRate)
	c.ptr.opts.audioSync = C.bool(o.AudioSync)
	c.ptr.opts.videoSync = C.bool(o.VideoSync)
	c.ptr.opts.volume = C.int(o.Volume)
	c.ptr.opts.fpsTarget = C.float(o.FPSTarget)
}

func (c *Core) Options() CoreOptions {
	return CoreOptions{
		AudioBuffers: int(c.ptr.opts.audioBuffers),
		SampleRate:   int(c.ptr.opts.sampleRate),
		AudioSync:    bool(c.ptr.opts.audioSync),
		VideoSync:    bool(c.ptr.opts.videoSync),
		Volume:       int(c.ptr.opts.volume),
		FPSTarget:    float32(c.ptr.opts.fpsTarget),
	}
}

func (c *Core) DesiredVideoDimensions() (int, int) {
	var width C.uint
	var height C.uint
	C.tango_mgba_mCore_desiredVideoDimensions(c.ptr, &width, &height)
	return int(width), int(height)
}

func (c *Core) SetVideoBuffer(buf unsafe.Pointer, width int) {
	C.tango_mgba_mCore_setVideoBuffer(c.ptr, (*C.uint)(buf), C.size_t(width))
}

func (c *Core) LoadROM(vf *VFile) error {
	if !C.tango_mgba_mCore_loadROM(c.ptr, vf.ptr) {
		return errors.New("could not load rom")
	}
	return nil
}

func (c *Core) LoadSave(vf *VFile) error {
	if !C.tango_mgba_mCore_loadSave(c.ptr, vf.ptr) {
		return errors.New("could not load save")
	}
	return nil
}

func (c *Core) GameTitle() string {
	var title [12]byte
	C.tango_mgba_mCore_getGameTitle(c.ptr, (*C.char)(unsafe.Pointer(&title)))
	return string(bytes.TrimRight(title[:], "\x00"))
}

func (c *Core) GameCode() string {
	var code [8]byte
	C.tango_mgba_mCore_getGameCode(c.ptr, (*C.char)(unsafe.Pointer(&code)))
	return string(code[:])
}

func (c *Core) Config() *Config {
	return c.config
}

func (c *Core) Reset() {
	C.tango_mgba_mCore_reset(c.ptr)
}

func (c *Core) RunFrame() {
	C.tango_mgba_mCore_runFrame(c.ptr)
}

func (c *Core) Frequency() int32 {
	return int32(C.tango_mgba_mCore_frequency(c.ptr))
}

func (c *Core) FrameCounter() uint32 {
	return uint32(C.tango_mgba_mCore_frameCounter(c.ptr))
}

func (c *Core) AudioBufferSize() int {
	return int(C.tango_mgba_mCore_getAudioBufferSize(c.ptr))
}

func (c *Core) SetAudioBufferSize(samples int) {
	C.tango_mgba_mCore_setAudioBufferSize(c.ptr, C.size_t(samples))
}

func (c *Core) AudioChannel(ch int) *Blip {
	return &Blip{C.tango_mgba_mCore_getAudioChannel(c.ptr, C.int(ch))}
}

func (c *Core) SetSync(sync *Sync) {
	C.tango_mgba_mCore_setSync(c.ptr, sync.ptr)
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
	c.config.Deinit()
	C.tango_mgba_mCore_deinit(c.ptr)
	c.ptr = nil
}

func (c *Core) CRC32() uint32 {
	var data [4]byte
	C.tango_mgba_mCore_checksum(c.ptr, unsafe.Pointer(&data), C.mCHECKSUM_CRC32)
	return uint32(*(*C.uint32_t)(unsafe.Pointer(&data)))
}
