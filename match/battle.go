package match

import (
	"context"

	"github.com/murkland/bbn6/input"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/replay"
	"github.com/murkland/ringbuf"
)

type Battle struct {
	isP2 bool

	rw *replay.Writer

	localInputBuffer *ringbuf.RingBuf[uint16]

	iq *input.Queue

	remoteInitCh chan []byte

	localPendingTurnWaitTicksLeft int
	localPendingTurn              []byte

	lastCommittedRemoteInput input.Input

	committedState *mgba.State
}

func (b *Battle) LocalPlayerIndex() int {
	if b.isP2 {
		return 1
	}
	return 0
}

func (b *Battle) RemotePlayerIndex() int {
	return 1 - b.LocalPlayerIndex()
}

func (b *Battle) Close() error {
	if err := b.rw.Close(); err != nil {
		return err
	}
	return nil
}

func (b *Battle) SetCommittedState(state *mgba.State) {
	b.committedState = state
}

func (b *Battle) CommittedState() *mgba.State {
	return b.committedState
}

func (b *Battle) ReadRemoteInit(ctx context.Context) ([]byte, error) {
	select {
	case init := <-b.remoteInitCh:
		return init, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *Battle) ConsumeInputs() ([][2]input.Input, []input.Input) {
	inputPairs := b.iq.Consume()
	if len(inputPairs) > 0 {
		b.lastCommittedRemoteInput = inputPairs[len(inputPairs)-1][1-b.LocalPlayerIndex()]
	}
	left := b.iq.Peek(b.LocalPlayerIndex())
	return inputPairs, left
}

func (b *Battle) AddInput(ctx context.Context, playerIndex int, input input.Input) error {
	return b.iq.AddInput(ctx, playerIndex, input)
}

func (b *Battle) AddLocalBufferedInputAndConsume(nextJoyflags uint16) uint16 {
	joyflags := uint16(0xfc00)
	if b.localInputBuffer.Free() == 0 {
		var joyflagsBuf [1]uint16
		b.localInputBuffer.Pop(joyflagsBuf[:], 0)
		joyflags = joyflagsBuf[0]
	}
	b.localInputBuffer.Push([]uint16{nextJoyflags})
	return joyflags
}

func (b *Battle) AddLocalPendingTurn(turn []byte) {
	b.localPendingTurn = turn
	b.localPendingTurnWaitTicksLeft = 64
}

func (b *Battle) ConsumeLocalPendingTurn() []byte {
	var turn []byte
	if b.localPendingTurnWaitTicksLeft > 0 {
		b.localPendingTurnWaitTicksLeft--
		if b.localPendingTurnWaitTicksLeft == 0 {
			turn = b.localPendingTurn
			b.localPendingTurn = nil
		}
	}
	return turn
}

func (b *Battle) LastCommittedRemoteInput() input.Input {
	return b.lastCommittedRemoteInput
}

func (b *Battle) ReplayWriter() *replay.Writer {
	return b.rw
}

func (b *Battle) Lag() int {
	return b.iq.Lag(b.RemotePlayerIndex())
}

func (b *Battle) IsP2() bool {
	return b.isP2
}
