package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/clone"
	"github.com/murkland/ctxwebrtc"
	signorclient "github.com/murkland/signor/client"
	"github.com/pion/webrtc/v3"
)

var defaultWebRTCConfig = (func() string {
	s, err := json.Marshal(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun1.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun2.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun3.l.google.com:19302"},
			},
			{
				URLs: []string{"stun:stun4.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return string(s)
})()

var (
	connectAddr  = flag.String("connect_addr", "http://localhost:12345", "address to connect to")
	answer       = flag.Bool("answer", false, "if true, answers a session instead of offers")
	sessionID    = flag.String("session_id", "test-session", "session to join to")
	webrtcConfig = flag.String("webrtc_config", defaultWebRTCConfig, "webrtc configuration")
	romPath      = flag.String("rom_path", "bn6f.gba", "path to rom")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	mgba.SetDefaultLogger(func(category string, level int, message string) {
		if level&0x7 == 0 {
			return
		}
		log.Printf("mgba: level=%d category=%s %s", level, category, message)
	})

	var peerConnConfig webrtc.Configuration
	if err := json.Unmarshal([]byte(*webrtcConfig), &peerConnConfig); err != nil {
		log.Fatalf("failed to parse webrtc config: %s", err)
	}

	log.Printf("connecting to %s, answer = %t, session_id = %s (using peer config: %+v)", *connectAddr, *answer, *sessionID, peerConnConfig)

	signorClient := signorclient.New(*connectAddr)

	peerConn, err := webrtc.NewPeerConnection(peerConnConfig)
	if err != nil {
		log.Fatalf("failed to create RTC peer connection: %s", err)
	}

	rtcDc, err := peerConn.CreateDataChannel("game", &webrtc.DataChannelInit{
		ID:         clone.P(uint16(1)),
		Negotiated: clone.P(true),
		Ordered:    clone.P(true),
	})
	if err != nil {
		log.Fatalf("failed to create RTC peer connection: %s", err)
	}

	dc := ctxwebrtc.WrapDataChannel(rtcDc)

	if !*answer {
		if err := signorClient.Offer(ctx, []byte(*sessionID), peerConn); err != nil {
			log.Fatalf("failed to offer: %s", err)
		}
	} else {
		if err := signorClient.Answer(ctx, []byte(*sessionID), peerConn); err != nil {
			log.Fatalf("failed to answer: %s", err)
		}
	}

	log.Printf("signaling complete!")
	log.Printf("local SDP: %s", peerConn.LocalDescription().SDP)
	log.Printf("remote SDP: %s", peerConn.RemoteDescription().SDP)

	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowTitle("bbn6")
	ebiten.SetMaxTPS(ebiten.UncappedTPS)
	ebiten.SetWindowResizable(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	g, err := game.New(*romPath, dc, *answer)
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
