package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/packets"
	"github.com/murkland/clone"
	"github.com/murkland/ctxwebrtc"
	signorclient "github.com/murkland/signor/client"
	"github.com/murkland/syncrand"
	"github.com/pion/webrtc/v3"
)

var (
	connectAddr = flag.String("connect_addr", "localhost:12345", "address to connect to")
	sessionID   = flag.String("session_id", "test-session", "session to join to")
	configPath  = flag.String("config_path", "bbn6.toml", "path to config")
	romPath     = flag.String("rom_path", "bn6.gba", "path to rom")
)

var commitHash string

const protocolVersion = 0x02

func Negotiate(ctx context.Context, dc *ctxwebrtc.DataChannel) (*syncrand.Source, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate rng seed part: %w", err)
	}

	commitment := syncrand.Commit(nonce[:])
	var helloPacket packets.Hello
	helloPacket.ProtocolVersion = protocolVersion
	copy(helloPacket.RNGCommitment[:], commitment)
	if err := packets.Send(ctx, dc, helloPacket, nil); err != nil {
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	theirHello, _, err := packets.Recv(ctx, dc)
	if err != nil {
		return nil, fmt.Errorf("failed to receive hello: %w", err)
	}
	if theirHello.(packets.Hello).ProtocolVersion != protocolVersion {
		return nil, fmt.Errorf("expected protocol version 0x%02x, got 0x%02x: are you out of date?", protocolVersion, protocolVersion)
	}

	theirCommitment := theirHello.(packets.Hello).RNGCommitment

	if err := packets.Send(ctx, dc, packets.Hello2{RNGNonce: nonce}, nil); err != nil {
		return nil, fmt.Errorf("failed to send hello2: %w", err)
	}

	theirHello2, _, err := packets.Recv(ctx, dc)
	if err != nil {
		return nil, fmt.Errorf("failed to receive hello2: %w", err)
	}
	theirNonce := theirHello2.(packets.Hello2).RNGNonce

	if !syncrand.Verify(commitment, theirCommitment[:], theirNonce[:]) {
		return nil, errors.New("failed to verify rng commitment")
	}

	seed := syncrand.MakeSeed(nonce[:], theirNonce[:])
	rng := syncrand.NewSource(seed)

	return rng, nil
}

func main() {
	flag.Parse()
	ctx := context.Background()

	log.Printf("welcome to bingus battle network 6. commit hash = %s", commitHash)

	var conf config.Config
	confF, err := os.Open(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config doesn't exist, making a new one at: %s", *configPath)
			confF, err = os.Create(*configPath)
			if err != nil {
				log.Fatalf("failed to open config: %s", err)
			}
			conf = config.DefaultConfig
			if err := config.Save(conf, confF); err != nil {
				log.Fatalf("failed to save config: %s", err)
			}
		} else {
			log.Fatalf("failed to open config: %s", err)
		}
	} else {
		conf, err = config.Load(confF)
		if err != nil {
			log.Fatalf("failed to open config: %s", err)
		}
		confF.Close()
	}

	log.Printf("config settings: %+v", conf.ToRaw())

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	log.Printf("connecting to %s, session_id = %s", *connectAddr, *sessionID)

	signorClient, err := signorclient.New(*connectAddr)
	if err != nil {
		log.Fatalf("failed to open matchmaking client: %s", err)
	}

	var rtcDc *webrtc.DataChannel
	peerConn, connectionSide, err := signorClient.Connect(ctx, *sessionID, func() (*webrtc.PeerConnection, error) {
		peerConn, err := webrtc.NewPeerConnection(conf.WebRTC)
		if err != nil {
			log.Fatalf("failed to create RTC peer connection: %s", err)
		}

		rtcDc, err = peerConn.CreateDataChannel("game", &webrtc.DataChannelInit{
			ID:         clone.P(uint16(1)),
			Negotiated: clone.P(true),
			Ordered:    clone.P(true),
		})
		if err != nil {
			log.Fatalf("failed to create RTC peer connection: %s", err)
		}

		return peerConn, nil
	})
	if err != nil {
		log.Fatalf("failed to connect to peer: %s", err)
	}
	dc := ctxwebrtc.WrapDataChannel(rtcDc)

	log.Printf("signaling complete!")
	log.Printf("local SDP: %s", peerConn.LocalDescription().SDP)
	log.Printf("remote SDP: %s", peerConn.RemoteDescription().SDP)

	randSource, err := Negotiate(ctx, dc)
	if err != nil {
		log.Fatalf("failed to negotiate connection with remote: %s", err)
	}
	log.Printf("connection negotiation ok!")

	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowTitle("bbn6")
	ebiten.SetMaxTPS(ebiten.UncappedTPS)
	ebiten.SetWindowResizable(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	g, err := game.New(conf, *romPath, dc, randSource, connectionSide)
	if err != nil {
		log.Fatalf("failed to start game: %s", err)
	}

	go func() {
		if err := g.RunBackgroundTasks(ctx); err != nil {
			log.Fatalf("error running background tasks: %s", err)
		}
	}()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("failed to run mgba: %s", err)
	}

	g.Finish()
}
