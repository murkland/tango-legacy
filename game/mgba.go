package game

import "github.com/murkland/bbn6/mgba"

var coreOptions = mgba.CoreOptions{
	SampleRate:   48000,
	AudioBuffers: 1024,
	AudioSync:    true,
	VideoSync:    true,
	Volume:       0x80,
}

func newCore(romPath string) (*mgba.Core, error) {
	core, err := mgba.FindCore(romPath)
	if err != nil {
		return nil, err
	}
	core.SetOptions(coreOptions)

	if err := core.LoadFile(romPath); err != nil {
		return nil, err
	}

	core.Config().Init("bbn6")
	core.Config().Load()
	core.LoadConfig()

	return core, nil
}
