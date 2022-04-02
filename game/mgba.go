package game

import (
	"errors"
	"os"

	"github.com/murkland/tango/mgba"
)

var coreOptions = mgba.CoreOptions{
	SampleRate:   48000,
	AudioBuffers: 1024,
	AudioSync:    true,
	VideoSync:    true,
	Volume:       0x100,
	FPSTarget:    60,
}

func newCore(romPath string) (*mgba.Core, error) {
	core, err := mgba.NewGBACore()
	if err != nil {
		return nil, err
	}
	core.SetOptions(coreOptions)

	vf := mgba.OpenVF(romPath, os.O_RDONLY)
	if vf == nil {
		return nil, errors.New("failed to open file")
	}

	if err := core.LoadROM(vf); err != nil {
		return nil, err
	}

	core.Config().Init("tango")
	core.Config().Load()
	core.SetAudioBufferSize(coreOptions.AudioBuffers)

	return core, nil
}
