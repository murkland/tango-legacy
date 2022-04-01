package av

import (
	"io"
	"runtime"
	"unsafe"

	"github.com/murkland/bbn6/mgba"
)

/*
#include <stdlib.h>
*/
import "C"

type ClippyAudioReader struct {
	core       *mgba.Core
	sampleRate int
	buf        unsafe.Pointer
}

func (a *ClippyAudioReader) Read(p []byte) (int, error) {
	p = p[:a.core.AudioBufferSize()*2*2]
	sync := a.core.GBA().Sync()

	fauxClock := float32(1)
	if sync != nil {
		fauxClock = mgba.GBAAudioCalculateRatio(1, sync.FPSTarget(), 1)
	}

	left := a.core.AudioChannel(0)
	right := a.core.AudioChannel(1)
	clockRate := a.core.Frequency()

	if sync != nil {
		sync.LockAudio()
	}

	left.SetRates(float64(clockRate), float64(a.sampleRate))
	right.SetRates(float64(clockRate), float64(a.sampleRate))

	audioBufStretchedBytesSize := ((int(float32(len(p))*fauxClock)+1)/2 + 1) / 2 * 2 * 2
	n := audioBufStretchedBytesSize / (2 * 2)
	available := left.SamplesAvail()
	if available > n {
		available = n
	}

	left.ReadSamples(a.buf, available, true)
	right.ReadSamples(unsafe.Add(a.buf, 2), available, true)
	copy(p, C.GoBytes(a.buf, C.int(available*2*2)))

	if sync != nil {
		sync.ConsumeAudio()
	}

	return len(p), nil
}

func NewClippyAudioReader(core *mgba.Core, sampleRate int) io.Reader {
	audioBufRealtimeBytesSize := core.AudioBufferSize() * 2 * 2
	// Use 4x buffer size to accommodate audio stretching up to 4x in time.
	buf := C.calloc(1, C.size_t(audioBufRealtimeBytesSize*4))
	ar := &ClippyAudioReader{core, sampleRate, buf}
	runtime.SetFinalizer(ar, func(ar *ClippyAudioReader) {
		C.free(ar.buf)
	})
	return ar
}
