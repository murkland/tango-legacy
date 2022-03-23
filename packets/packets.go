package packets

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"io"
	"log"

	"github.com/murkland/ctxwebrtc"
)

var (
	debugLogPackets = flag.Bool("debug_log_packets", false, "log all packets (noisy!)")
)

var ErrUnknownPacket = errors.New("unknown packet")

type packetType uint8

const (
	packetTypePing  packetType = 0
	packetTypePong  packetType = 1
	packetTypeInit  packetType = 2
	packetTypeTurn  packetType = 3
	packetTypeInput packetType = 4
)

type Packet interface {
	packetType() packetType
}

type Ping struct {
	ID uint64
}

func (Ping) packetType() packetType { return packetTypePing }

type Pong struct {
	ID uint64
}

func (Pong) packetType() packetType { return packetTypePong }

type Init struct {
	Marshaled [0x100]uint8
}

func (Init) packetType() packetType { return packetTypeInit }

type Turn struct {
	ForTick   uint32
	Marshaled [0x100]uint8
}

func (Turn) packetType() packetType { return packetTypeTurn }

type Input struct {
	ForTick           uint32
	Joyflags          uint16
	CustomScreenState uint8
}

func (Input) packetType() packetType { return packetTypeInput }

func Marshal(packet Packet) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, packet.packetType()); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, packet); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func unmarshal[T Packet](r io.Reader) (T, error) {
	var packet T
	if err := binary.Read(r, binary.LittleEndian, &packet); err != nil {
		return packet, err
	}
	return packet, nil
}

func Unmarshal(raw []byte) (Packet, error) {
	r := bytes.NewReader(raw)
	var typ packetType
	if err := binary.Read(r, binary.LittleEndian, &typ); err != nil {
		return nil, err
	}

	switch typ {
	case packetTypePing:
		return unmarshal[Ping](r)
	case packetTypePong:
		return unmarshal[Pong](r)
	case packetTypeInit:
		return unmarshal[Init](r)
	case packetTypeTurn:
		return unmarshal[Turn](r)
	case packetTypeInput:
		return unmarshal[Input](r)
	default:
		return nil, ErrUnknownPacket
	}
}

func Send(ctx context.Context, dc *ctxwebrtc.DataChannel, packet Packet) error {
	if *debugLogPackets {
		log.Printf("--> %#v", packet)
	}
	return dc.Send(ctx, Marshal(packet))
}

func Recv(ctx context.Context, dc *ctxwebrtc.DataChannel) (Packet, error) {
	raw, err := dc.Recv(ctx)
	if err != nil {
		return nil, err
	}
	packet, err := Unmarshal(raw)
	if err != nil {
		return nil, err
	}
	if *debugLogPackets {
		log.Printf("<-- %#v", packet)
	}
	return packet, nil
}
