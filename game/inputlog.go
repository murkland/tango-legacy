package game

type InputLog struct {
}

func newInputLog() (*InputLog, error) {
	return &InputLog{}, nil
}

func (il *InputLog) WriteInit(playerIndex int, marshaled []byte) error {
	return nil
}

func (il *InputLog) Write(rngState uint32, inputPair [2]Input) error {
	p1 := inputPair[0]
	p2 := inputPair[1]
	_ = p1
	_ = p2

	return nil
}
