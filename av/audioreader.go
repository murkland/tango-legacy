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
	p = p[:a.core.AudioBufferSize()*2*2]

	sync := a.core.GBA().Sync()

	fauxClock := float32(1)
	if sync != nil {
		fauxClock = mgba.GBAAudioCalculateRatio(1, sync.FPSTarget(), 1)
	}

	audioBufStretchedBytesSize := (int(float32(len(p))*fauxClock) + 1) / 2 * 2

	left := a.core.AudioChannel(0)
	right := a.core.AudioChannel(1)
	clockRate := a.core.Frequency()

	if sync != nil {
		sync.LockAudio()
	}

	left.SetRates(float64(clockRate), float64(a.sampleRate))
	right.SetRates(float64(clockRate), float64(a.sampleRate))

	available := left.SamplesAvail()
	if available > audioBufStretchedBytesSize {
		available = audioBufStretchedBytesSize
	}

	left.ReadSamples(a.buf, available, true)
	right.ReadSamples(unsafe.Pointer(uintptr(a.buf)+2), available, true)
	copy(p, C.GoBytes(a.buf, C.int(audioBufStretchedBytesSize)))

	if sync != nil {
		sync.ConsumeAudio()
	}

	return len(p), nil
}

func NewAudioReader(core *mgba.Core, sampleRate int) *AudioReader {
	audioBufRealtimeBytesSize := core.AudioBufferSize() * 2 * 2
	buf := C.calloc(1, C.size_t(audioBufRealtimeBytesSize*2))
	ar := &AudioReader{core, sampleRate, buf}
	runtime.SetFinalizer(ar, func(ar *AudioReader) {
		C.free(ar.buf)
	})
	return ar
}
