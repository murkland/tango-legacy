package packets

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"io"
	"io/ioutil"
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
	packetTypeInput packetType = 3
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

// Input has an occasional 256 byte trailer.
type Input struct {
	ForTick           uint32
	Joyflags          uint16
	CustomScreenState uint8
}

func (Input) packetType() packetType { return packetTypeInput }

func Marshal(packet Packet, w io.Writer) {
	if err := binary.Write(w, binary.LittleEndian, packet.packetType()); err != nil {
		panic(err)
	}
	if err := binary.Write(w, binary.LittleEndian, packet); err != nil {
		panic(err)
	}
}

func unmarshal[T Packet](r io.Reader) (T, error) {
	var packet T
	if err := binary.Read(r, binary.LittleEndian, &packet); err != nil {
		return packet, err
	}
	return packet, nil
}

func Unmarshal(r io.Reader) (Packet, error) {
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
	case packetTypeInput:
		return unmarshal[Input](r)
	default:
		return nil, ErrUnknownPacket
	}
}

func Send(ctx context.Context, dc *ctxwebrtc.DataChannel, packet Packet, trailer []byte) error {
	if *debugLogPackets {
		log.Printf("--> %#v trailer=%v", packet, trailer)
	}
	var buf bytes.Buffer
	Marshal(packet, &buf)
	buf.Write(trailer)
	return dc.Send(ctx, buf.Bytes())
}

func Recv(ctx context.Context, dc *ctxwebrtc.DataChannel) (Packet, []byte, error) {
	raw, err := dc.Recv(ctx)
	if err != nil {
		return nil, nil, err
	}
	r := bytes.NewReader(raw)
	packet, err := Unmarshal(r)
	if err != nil {
		return nil, nil, err
	}
	trailer, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	if len(trailer) == 0 {
		trailer = nil
	}
	if *debugLogPackets {
		log.Printf("<-- %#v trailer=%v", packet, trailer)
	}
	return packet, trailer, nil
}
