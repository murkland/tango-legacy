package input

type Input struct {
	LocalTick               int
	LastCommittedRemoteTick int
	Joyflags                uint16
	CustomScreenState       uint8
	Turn                    []byte
}
