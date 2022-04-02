package input

type Input struct {
	LocalTick         int
	RemoteTick        int
	Joyflags          uint16
	CustomScreenState uint8
	Turn              []byte
}
