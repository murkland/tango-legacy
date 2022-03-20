package packets

type Type int

const (
	TypeInput Type = 1
)

type Input struct {
	Joyflags uint16
}
