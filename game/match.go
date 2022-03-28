package game

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keegancsmith/nth"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/packets"
	"github.com/murkland/clone"
	"github.com/murkland/ctxwebrtc"
	"github.com/murkland/ringbuf"
	signorclient "github.com/murkland/signor/client"
	"github.com/murkland/syncrand"
	"github.com/pion/webrtc/v3"
	"golang.org/x/sync/errgroup"
)

type Match struct {
	conf      config.Config
	sessionID string
	matchType uint8

	cancel context.CancelFunc

	connReady     chan struct{}
	peerConn      *webrtc.PeerConnection
	dc            *ctxwebrtc.DataChannel
	wonLastBattle bool
	randSource    rand.Source

	delayRingbuf   *ringbuf.RingBuf[time.Duration]
	delayRingbufMu sync.RWMutex

	stalledFrames int

	battleNumber int
	battle       *Battle
}

func NewMatch(conf config.Config, sessionID string, matchType uint8) (*Match, error) {
	return &Match{
		conf:      conf,
		sessionID: sessionID,
		matchType: matchType,

		connReady: make(chan struct{}),

		delayRingbuf: ringbuf.New[time.Duration](9),
	}, nil
}

func (m *Match) negotiate(ctx context.Context) error {
	log.Printf("connecting to %s, session_id = %s", m.conf.Matchmaking.ConnectAddr, m.sessionID)

	signorClient, err := signorclient.New(m.conf.Matchmaking.ConnectAddr)
	if err != nil {
		return err
	}

	var rtcDc *webrtc.DataChannel
	peerConn, connectionSide, err := signorClient.Connect(ctx, m.sessionID, func() (*webrtc.PeerConnection, error) {
		peerConn, err := webrtc.NewPeerConnection(m.conf.WebRTC)
		if err != nil {
			return nil, err
		}

		rtcDc, err = peerConn.CreateDataChannel("game", &webrtc.DataChannelInit{
			ID:         clone.P(uint16(1)),
			Negotiated: clone.P(true),
			Ordered:    clone.P(true),
		})
		if err != nil {
			return nil, err
		}

		return peerConn, nil
	})
	if err != nil {
		return err
	}
	dc := ctxwebrtc.WrapDataChannel(rtcDc)

	log.Printf("signaling complete!")
	log.Printf("local SDP: %s", peerConn.LocalDescription().SDP)
	log.Printf("remote SDP: %s", peerConn.RemoteDescription().SDP)

	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("failed to generate rng seed part: %w", err)
	}

	commitment := syncrand.Commit(nonce[:])
	var helloPacket packets.Hello
	helloPacket.ProtocolVersion = packets.ProtocolVersion
	helloPacket.MatchType = m.matchType
	copy(helloPacket.RNGCommitment[:], commitment)
	if err := packets.Send(ctx, dc, helloPacket, nil); err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}

	rawTheirHello, _, err := packets.Recv(ctx, dc)
	if err != nil {
		return fmt.Errorf("failed to receive hello: %w", err)
	}
	theirHello := rawTheirHello.(packets.Hello)
	if theirHello.ProtocolVersion != packets.ProtocolVersion {
		return fmt.Errorf("expected protocol version 0x%02x, got 0x%02x: are you out of date?", packets.ProtocolVersion, theirHello.ProtocolVersion)
	}

	if theirHello.MatchType != m.matchType {
		return errors.New("match type mismatch")
	}

	theirCommitment := theirHello.RNGCommitment

	if err := packets.Send(ctx, dc, packets.Hello2{RNGNonce: nonce}, nil); err != nil {
		return fmt.Errorf("failed to send hello2: %w", err)
	}

	theirHello2, _, err := packets.Recv(ctx, dc)
	if err != nil {
		return fmt.Errorf("failed to receive hello2: %w", err)
	}
	theirNonce := theirHello2.(packets.Hello2).RNGNonce

	if !syncrand.Verify(commitment, theirCommitment[:], theirNonce[:]) {
		return errors.New("failed to verify rng commitment")
	}

	seed := syncrand.MakeSeed(nonce[:], theirNonce[:])
	randSource := syncrand.NewSource(seed)

	m.peerConn = peerConn
	m.dc = dc
	m.randSource = randSource
	rng := rand.New(m.randSource)
	m.wonLastBattle = (rng.Int31n(2) == 1) == (connectionSide == signorclient.ConnectionSideOfferer)
	close(m.connReady)
	log.Printf("negotiation complete!")
	return nil
}

func (m *Match) medianDelay() time.Duration {
	m.delayRingbufMu.RLock()
	defer m.delayRingbufMu.RUnlock()

	if m.delayRingbuf.Used() == 0 {
		return 0
	}

	delays := make([]time.Duration, m.delayRingbuf.Used())
	m.delayRingbuf.Peek(delays, 0)

	i := len(delays) / 2
	nth.Element(orderableSlice[time.Duration](delays), i)
	return delays[i]
}

func (m *Match) sendPings(ctx context.Context) error {
	for {
		now := time.Now()
		if err := packets.Send(ctx, m.dc, packets.Ping{
			ID: uint64(now.UnixMicro()),
		}, nil); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func (m *Match) handleConn(ctx context.Context) error {
	for {
		packet, trailer, err := packets.Recv(ctx, m.dc)
		if err != nil {
			return err
		}

		switch p := packet.(type) {
		case packets.Ping:
			if err := packets.Send(ctx, m.dc, packets.Pong{ID: p.ID}, nil); err != nil {
				return err
			}
		case packets.Pong:
			if err := (func() error {
				m.delayRingbufMu.Lock()
				defer m.delayRingbufMu.Unlock()

				if m.delayRingbuf.Free() == 0 {
					m.delayRingbuf.Advance(1)
				}

				delay := time.Now().Sub(time.UnixMicro(int64(p.ID)))
				m.delayRingbuf.Push([]time.Duration{delay})
				return nil
			})(); err != nil {
				return err
			}
		case packets.Init:
			m.battle.remoteInit = p.Marshaled[:]
		case packets.Input:
			if m == nil || m.battle == nil {
				log.Printf("received input packet while battle was apparently not active, dropping it (this may cause a desync!)")
				continue
			}
			m.battle.iq.AddInput(ctx, m.battle.RemotePlayerIndex(), Input{int(p.ForTick), p.Joyflags, p.CustomScreenState, trailer})
		}
	}
}

func (m *Match) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	defer m.Close()

	if err := m.negotiate(ctx); err != nil {
		return err
	}

	errg, ctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return m.handleConn(ctx)
	})
	errg.Go(func() error {
		return m.sendPings(ctx)
	})
	return errg.Wait()
}

func (m *Match) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	if m.battle != nil {
		if err := m.battle.Close(); err != nil {
			return err
		}
	}
	if m.dc != nil {
		if err := m.dc.Close(); err != nil {
			return err
		}
	}
	if m.peerConn != nil {
		if err := m.peerConn.Close(); err != nil {
			return err
		}
	}
	return nil
}

type Battle struct {
	tick int32
	isP2 bool

	rw *ReplayWriter

	localInputBuffer *ringbuf.RingBuf[uint16]

	iq *InputQueue

	localInit      []byte
	remoteInit     []byte
	remoteInitUsed bool

	localPendingTurnWaitTicksLeft int
	localPendingTurn              []byte

	lastCommittedRemoteInput Input

	committedState *mgba.State
}

func (s *Battle) LocalPlayerIndex() int {
	if s.isP2 {
		return 1
	}
	return 0
}

func (s *Battle) RemotePlayerIndex() int {
	return 1 - s.LocalPlayerIndex()
}

func NewBattle(core *mgba.Core, isP2 bool, localInputBufferSize int) (*Battle, error) {
	b := &Battle{
		isP2: isP2,

		lastCommittedRemoteInput: Input{Joyflags: 0xfc00},

		localInputBuffer: ringbuf.New[uint16](localInputBufferSize),

		iq: NewInputQueue(60),
	}

	os.MkdirAll("replays", 0o700)
	fn := filepath.Join("replays", fmt.Sprintf("%s_p%d.bbn6replay", time.Now().Format("20060102030405"), b.LocalPlayerIndex()+1))
	log.Printf("writing replay: %s", fn)

	il, err := newReplayWriter(fn, core)
	if err != nil {
		return nil, err
	}
	b.rw = il
	return b, nil
}

func (s *Battle) Close() error {
	if err := s.rw.Close(); err != nil {
		return err
	}
	return nil
}
