package input

type Input struct {
	Tick              int
	Joyflags          uint16
	CustomScreenState uint8
	Turn              []byte
}
