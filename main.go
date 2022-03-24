package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/game"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/clone"
	"github.com/murkland/ctxwebrtc"
	signorclient "github.com/murkland/signor/client"
	"github.com/pion/webrtc/v3"
)

var (
	connectAddr = flag.String("connect_addr", "http://localhost:12345", "address to connect to")
	answer      = flag.Bool("answer", false, "if true, answers a session instead of offers")
	sessionID   = flag.String("session_id", "test-session", "session to join to")
	configPath  = flag.String("config_path", "bn6f.toml", "path to config")
	romPath     = flag.String("rom_path", "bn6f.gba", "path to rom")
)

var commitHash string

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
			if err := config.Save(config.DefaultConfig, confF); err != nil {
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

	log.Printf("connecting to %s, answer = %t, session_id = %s", *connectAddr, *answer, *sessionID)

	signorClient := signorclient.New(*connectAddr)

	peerConn, err := webrtc.NewPeerConnection(conf.WebRTC)
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

	g, err := game.New(conf, *romPath, dc, *answer)
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
