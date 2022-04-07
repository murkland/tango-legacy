package input

import (
	"context"
	"sync"

	"github.com/murkland/ringbuf"
)

type Queue struct {
	mu   sync.Mutex
	cond *sync.Cond

	localPlayerIndex int
	qs               [2]*ringbuf.RingBuf[Input]
	localDelay       int
}

func NewQueue(n int, localDelay int, localPlayerIndex int) *Queue {
	iq := &Queue{
		localPlayerIndex: localPlayerIndex,
		qs: [2]*ringbuf.RingBuf[Input]{
			ringbuf.New[Input](n),
			ringbuf.New[Input](n),
		},
		localDelay: localDelay,
	}
	iq.cond = sync.NewCond(&iq.mu)
	return iq
}

func (q *Queue) AddInput(ctx context.Context, playerIndex int, input Input) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var ctxErr error
	go func() {
		<-ctx.Done()
		ctxErr = ctx.Err()
		q.cond.Broadcast()
	}()

	for q.qs[playerIndex].Free() == 0 && ctxErr == nil {
		q.cond.Wait()
	}
	if ctxErr != nil {
		return ctxErr
	}

	q.qs[playerIndex].Push([]Input{input})
	return nil
}

func (q *Queue) QueueLength(playerIndex int) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.qs[playerIndex].Used()
}

func (q *Queue) advanceManyLocked() [][2]Input {
	n := q.qs[q.localPlayerIndex].Used() - q.localDelay
	if q.qs[1-q.localPlayerIndex].Used() < n {
		n = q.qs[1-q.localPlayerIndex].Used()
	}

	if n < 0 {
		return nil
	}

	p1Inputs := make([]Input, n)
	q.qs[0].Pop(p1Inputs, 0)

	p2Inputs := make([]Input, n)
	q.qs[1].Pop(p2Inputs, 0)

	inputPairs := make([][2]Input, n)
	for i := 0; i < n; i++ {
		inputPairs[i] = [2]Input{p1Inputs[i], p2Inputs[i]}
	}

	q.cond.Broadcast()
	return inputPairs
}

func (q *Queue) ConsumeAndPeekLocal() ([][2]Input, []Input) {
	q.mu.Lock()
	defer q.mu.Unlock()

	consumable := q.advanceManyLocked()

	n := q.qs[q.localPlayerIndex].Used() - q.localDelay
	if n < 0 {
		n = 0
	}
	inputs := make([]Input, n)
	q.qs[q.localPlayerIndex].Peek(inputs, 0)

	return consumable, inputs
}

func (q *Queue) LocalDelay() int {
	return q.localDelay
}
