package game

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/murkland/bbn6/mgba"
)

type Battle struct {
	mu sync.Mutex

	tick uint32
	isP2 bool

	inputlog *InputLog
	iq       *InputQueue

	localPendingTurnWaitTicksLeft int
	localPendingTurn              []byte

	lastCommittedRemoteInput Input

	committedState *mgba.State
}

func (s *Battle) LocalPlayerIndex() int {
	if s.isP2 {
		return 1
	}
	return 0
}

func (s *Battle) RemotePlayerIndex() int {
	return 1 - s.LocalPlayerIndex()
}

func NewBattle(isP2 bool) (*Battle, error) {
	b := &Battle{
		tick: 0,
		isP2: isP2,

		lastCommittedRemoteInput: Input{Joyflags: 0xfc00},

		iq: NewInputQueue(60),
	}

	fn := fmt.Sprintf("input_p%d_%s.log", b.LocalPlayerIndex()+1, time.Now().Format("20060102030405"))
	log.Printf("opening input log: %s", fn)

	il, err := newInputLog(fn)
	if err != nil {
		return nil, err
	}
	b.inputlog = il
	return b, nil
}

func (s *Battle) Close() error {
	if err := s.inputlog.Close(); err != nil {
		return err
	}
	return nil
}
