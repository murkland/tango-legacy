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

type RubberyAudioReader struct {
	core       *mgba.Core
	sampleRate int
	buf        unsafe.Pointer
}

func (a *RubberyAudioReader) Read(p []byte) (int, error) {
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

	left.SetRates(float64(clockRate), float64(a.sampleRate)*float64(fauxClock))
	right.SetRates(float64(clockRate), float64(a.sampleRate)*float64(fauxClock))

	n := len(p) / (2 * 2)
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

func NewRubberyAudioReader(core *mgba.Core, sampleRate int) io.Reader {
	buf := C.calloc(1, C.size_t(core.AudioBufferSize()*2*2))
	ar := &RubberyAudioReader{core, sampleRate, buf}
	runtime.SetFinalizer(ar, func(ar *RubberyAudioReader) {
		C.free(ar.buf)
	})
	return ar
}
