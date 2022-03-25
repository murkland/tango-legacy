package game

import (
	"sync"

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
	il, err := newInputLog()
	if err != nil {
		return nil, err
	}

	return &Battle{
		tick: 0,
		isP2: isP2,

		lastCommittedRemoteInput: Input{Joyflags: 0xfc00},

		inputlog: il,
		iq:       NewInputQueue(60),
	}, nil
}
