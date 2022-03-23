package mgba

/*
#include <mgba/core/blip_buf.h>
*/
import "C"
import "unsafe"

type Blip struct {
	ptr *C.struct_blip_t
}

func (b *Blip) SetRates(clockRate float64, sampleRate float64) {
	C.blip_set_rates(b.ptr, C.double(clockRate), C.double(sampleRate))
}

func (b *Blip) SamplesAvail() int {
	return int(C.blip_samples_avail(b.ptr))
}

func (b *Blip) ReadSamples(out unsafe.Pointer, count int, stereo int) int {
	return int(C.blip_read_samples(b.ptr, (*C.short)(out), C.int(count), C.int(stereo)))
}
