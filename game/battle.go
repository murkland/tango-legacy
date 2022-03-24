package game

import (
	"context"
	"fmt"

	"github.com/murkland/bbn6/packets"
	"github.com/murkland/ctxwebrtc"
	"github.com/murkland/ringbuf"
)

type turn struct {
	tick      int
	marshaled []byte
}

type Battle struct {
	StartFrameNumber uint32
	LastTick         int
	IsP2             bool

	initSent     bool
	initReceived bool

	localTurn  *turn
	remoteTurn *turn

	localInputQueue  *ringbuf.RingBuf[Input]
	remoteInputQueue *ringbuf.RingBuf[Input]
}

func (s *Battle) LocalPlayerIndex() int {
	if s.IsP2 {
		return 1
	}
	return 0
}

func (s *Battle) RemotePlayerIndex() int {
	return 1 - s.LocalPlayerIndex()
}

func NewBattle(isP2 bool) *Battle {
	const inputDelay = 6

	localInputQueue := ringbuf.New[Input](inputDelay + 1)
	remoteInputQueue := ringbuf.New[Input](inputDelay + 1)

	dummyInput := make([]Input, inputDelay)
	for i := 0; i < len(dummyInput); i++ {
		dummyInput[i] = Input{Tick: i - inputDelay, Joyflags: 0xfc00, CustomScreenState: 0}
	}

	localInputQueue.Push(dummyInput)
	remoteInputQueue.Push(dummyInput)

	return &Battle{
		0, -1, isP2,
		false, false,
		nil, nil,
		localInputQueue, remoteInputQueue,
	}
}

func (s *Battle) ReceiveInit(ctx context.Context, dc *ctxwebrtc.DataChannel) ([]byte, error) {
	if s.initReceived || !s.initSent {
		return nil, nil
	}

	pkt, err := packets.Recv(ctx, dc)
	if err != nil {
		return nil, err
	}

	initPkt, ok := pkt.(packets.Init)
	if !ok {
		return nil, fmt.Errorf("unexpected packet: %v", initPkt)
	}

	s.initReceived = true
	return initPkt.Marshaled[:], nil
}

func (s *Battle) SendInit(ctx context.Context, dc *ctxwebrtc.DataChannel, marshaled []byte) error {
	s.initSent = true
	var pkt packets.Init
	copy(pkt.Marshaled[:], marshaled)
	return packets.Send(ctx, dc, pkt)
}

func (s *Battle) QueueLocalTurn(ctx context.Context, dc *ctxwebrtc.DataChannel, tick int, marshaled []byte) error {
	s.localTurn = &turn{tick, marshaled}

	var pkt packets.Turn
	pkt.ForTick = uint32(tick)
	copy(pkt.Marshaled[:], marshaled)
	return packets.Send(ctx, dc, pkt)
}

func (s *Battle) QueueLocalInput(ctx context.Context, dc *ctxwebrtc.DataChannel, tick int, joyflags uint16, customScreenState uint8) error {
	if err := s.localInputQueue.Push([]Input{{tick, joyflags, customScreenState}}); err != nil {
		return err
	}

	var pkt packets.Input
	pkt.ForTick = uint32(tick)
	pkt.Joyflags = joyflags
	pkt.CustomScreenState = customScreenState
	return packets.Send(ctx, dc, pkt)
}

type Input struct {
	Tick              int
	Joyflags          uint16
	CustomScreenState uint8
}

type InputAndTurn struct {
	Input
	Turn []byte
}

func (s *Battle) DequeueInputs(ctx context.Context, dc *ctxwebrtc.DataChannel) (InputAndTurn, InputAndTurn, error) {
	for s.remoteInputQueue.Free() > 0 {
		pkt, err := packets.Recv(ctx, dc)
		if err != nil {
			return InputAndTurn{}, InputAndTurn{}, err
		}

		switch p := pkt.(type) {
		case packets.Turn:
			s.remoteTurn = &turn{tick: int(p.ForTick), marshaled: p.Marshaled[:]}
		case packets.Input:
			if err := s.remoteInputQueue.Push([]Input{{int(p.ForTick), p.Joyflags, p.CustomScreenState}}); err != nil {
				return InputAndTurn{}, InputAndTurn{}, err
			}
		default:
			return InputAndTurn{}, InputAndTurn{}, fmt.Errorf("unexpected packet: %v", p)
		}
	}

	var localInputBuf [1]Input
	if err := s.localInputQueue.Peek(localInputBuf[:], 0); err != nil {
		return InputAndTurn{}, InputAndTurn{}, err
	}
	if err := s.localInputQueue.Advance(1); err != nil {
		return InputAndTurn{}, InputAndTurn{}, err
	}

	var remoteInputBuf [1]Input
	if err := s.remoteInputQueue.Peek(remoteInputBuf[:], 0); err != nil {
		return InputAndTurn{}, InputAndTurn{}, err
	}
	if err := s.remoteInputQueue.Advance(1); err != nil {
		return InputAndTurn{}, InputAndTurn{}, err
	}

	local := InputAndTurn{Input: localInputBuf[0]}
	remote := InputAndTurn{Input: remoteInputBuf[0]}

	if s.localTurn != nil && s.localTurn.tick+1 == local.Tick {
		local.Turn = s.localTurn.marshaled
		s.localTurn = nil
	}

	if s.remoteTurn != nil && s.remoteTurn.tick+1 == remote.Tick {
		remote.Turn = s.remoteTurn.marshaled
		s.remoteTurn = nil
	}

	return local, remote, nil
}
