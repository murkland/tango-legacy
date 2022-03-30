package av

import (
	"runtime"
	"unsafe"

	"github.com/murkland/bbn6/mgba"
)

/*
#include <stdlib.h>
*/
import "C"

type AudioReader struct {
	core       *mgba.Core
	sampleRate int
	buf        unsafe.Pointer
}

func (a *AudioReader) Read(p []byte) (int, error) {
	left := a.core.AudioChannel(0)
	right := a.core.AudioChannel(1)
	clockRate := a.core.Frequency()

	sync := a.core.GBA().Sync()

	fauxClock := float32(1)
	if sync != nil {
		fauxClock = mgba.GBAAudioCalculateRatio(1, sync.FPSTarget(), 1)
	}

	realBufSize := a.core.Options().AudioBuffers * 2 * 2

	bufSize := int(float64(realBufSize) * float64(fauxClock))

	if sync != nil {
		sync.LockAudio()
	}

	left.SetRates(float64(clockRate), float64(a.sampleRate))
	right.SetRates(float64(clockRate), float64(a.sampleRate))

	available := left.SamplesAvail()
	if available > bufSize {
		available = bufSize
	}

	left.ReadSamples(a.buf, available, true)
	right.ReadSamples(unsafe.Pointer(uintptr(a.buf)+2), available, true)
	copy(p, C.GoBytes(a.buf, C.int(bufSize)))

	if sync != nil {
		sync.ConsumeAudio()
	}

	return realBufSize, nil
}

func NewAudioReader(core *mgba.Core, sampleRate int) *AudioReader {
	// bufsize is the expected buffer size for 16-bit 2 channel audio with AudioBuffers samples.
	bufSize := core.Options().AudioBuffers * 2 * 2

	// We use double the buffer size to handle speedups up to 200%.
	buf := C.calloc(1, C.size_t(bufSize*2))

	ar := &AudioReader{core, sampleRate, buf}
	runtime.SetFinalizer(ar, func(ar *AudioReader) {
		C.free(ar.buf)
	})
	return ar
}
