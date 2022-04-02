package input

type Input struct {
	Tick              int
	Lag               int8
	Joyflags          uint16
	CustomScreenState uint8
	Turn              []byte
}
