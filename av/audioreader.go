package av

import (
	"reflect"
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
}

func (a *AudioReader) Read(p []byte) (int, error) {
	p = p[:a.core.Options().AudioBuffers*2*2]

	left := a.core.AudioChannel(0)
	right := a.core.AudioChannel(1)
	clockRate := a.core.Frequency()

	sync := a.core.GBA().Sync()

	fauxClock := 1
	if sync != nil {
		mgba.GBAAudioCalculateRatio(1, sync.FPSTarget(), 1)
	}

	if sync != nil {
		sync.LockAudio()
	}

	left.SetRates(float64(clockRate), float64(a.sampleRate)*float64(fauxClock))
	right.SetRates(float64(clockRate), float64(a.sampleRate)*float64(fauxClock))

	available := left.SamplesAvail()
	if available > len(p) {
		available = len(p)
	}

	buf := make([]byte, len(p))
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	left.ReadSamples(unsafe.Pointer(sliceHeader.Data), available, true)
	right.ReadSamples(unsafe.Pointer(sliceHeader.Data+2), available, true)
	copy(p, buf)

	if sync != nil {
		sync.ConsumeAudio()
	}

	return len(p), nil
}

func NewAudioReader(core *mgba.Core, sampleRate int) *AudioReader {
	return &AudioReader{core, sampleRate}
}
