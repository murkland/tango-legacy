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
	InitReceived     bool
	IsP2             bool

	localTurn  *turn
	remoteTurn *turn

	localInputQueue  *ringbuf.RingBuf[Input]
	remoteInputQueue *ringbuf.RingBuf[Input]
}

func (s *Battle) PlayerIndex() int {
	if s.IsP2 {
		return 1
	}
	return 0
}

func NewBattle(isP2 bool) *Battle {
	const inputBufSize = 6

	return &Battle{
		0, -1, false, isP2,
		nil, nil,
		ringbuf.New[Input](inputBufSize), ringbuf.New[Input](inputBufSize),
	}
}

func (s *Battle) ReceiveInit(ctx context.Context, dc *ctxwebrtc.DataChannel) ([]byte, error) {
	if s.InitReceived {
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

	s.InitReceived = true
	return initPkt.Marshaled[:], nil
}

func (s *Battle) SendInit(ctx context.Context, dc *ctxwebrtc.DataChannel, marshaled []byte) error {
	var pkt packets.Init
	copy(pkt.Marshaled[:], marshaled)
	return packets.Send(ctx, dc, pkt)
}

func (s *Battle) QueueLocalInput(ctx context.Context, dc *ctxwebrtc.DataChannel, tick int, joyflags uint16, customScreenState uint8) error {
	s.localInputQueue.Push([]Input{{tick, joyflags, customScreenState}})

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
	var remoteInputBuf [1]Input
	if err := s.remoteInputQueue.Peek(remoteInputBuf[:], 0); err != nil {
		return InputAndTurn{}, InputAndTurn{}, err
	}

	local := InputAndTurn{Input: localInputBuf[0]}
	remote := InputAndTurn{Input: remoteInputBuf[0]}

	if s.localTurn != nil && s.localTurn.tick+1 == local.Tick {
		local.Turn = s.localTurn.marshaled
		local.Turn = nil
	}

	if s.remoteTurn != nil && s.remoteTurn.tick+1 == remote.Tick {
		remote.Turn = s.remoteTurn.marshaled
		remote.Turn = nil
	}

	return local, remote, nil
}

func (s *Battle) QueueLocalTurn(ctx context.Context, dc *ctxwebrtc.DataChannel, tick int, marshaled []byte) error {
	s.localTurn = &turn{tick, marshaled}

	var pkt packets.Turn
	pkt.ForTick = uint32(tick)
	copy(pkt.Marshaled[:], marshaled)
	return packets.Send(ctx, dc, pkt)
}
