package main

import (
	"encoding/binary"
	"flag"
	"log"
	"math/rand"
	"net"

	"github.com/murkland/bbn6/bbn6d/packets"
)

var (
	listenAddr = flag.String("listen_addr", "localhost:12345", "address to listen on")
)

func errMain() error {
	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		return err
	}

	log.Printf("listening on %s", lis.Addr())

	conn, err := lis.Accept()
	if err != nil {
		return err
	}

	log.Printf("accepted connection from client %s", conn.RemoteAddr())

	for {
		if _, err := conn.Write([]byte{byte(packets.TypeInput)}); err != nil {
			return err
		}

		joyflags := uint16(0xfc00)
		directions := []uint16{0x0010, 0x0020, 0x0040, 0x0080}
		joyflags |= directions[rand.Intn(len(directions))]

		if err := binary.Write(conn, binary.LittleEndian, packets.Input{Joyflags: joyflags}); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := errMain(); err != nil {
		log.Fatalf("errMain exited with error: %s", err)
	}
}
