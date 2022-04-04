package match

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/murkland/clone"
	"github.com/murkland/ctxwebrtc"
	signorclient "github.com/murkland/signor/client"
	"github.com/murkland/syncrand"
	"github.com/murkland/tango/config"
	"github.com/murkland/tango/input"
	"github.com/murkland/tango/packets"
	"github.com/pion/webrtc/v3"
)

const expectedFPS = 60

var (
	ErrNotReady                = errors.New("match not ready")
	ErrMatchTypeMismatch       = errors.New("match type mismatch")
	ErrGameTypeMismatch        = errors.New("game type mismatch (US vs JP)")
	ErrProtocolVersionMismatch = errors.New("protocol version mismatch")
)

type Match struct {
	conf      config.Config
	sessionID string
	matchType uint16
	gameTitle string
	gameCRC32 uint32

	cancel context.CancelFunc

	negotiationErrCh chan error
	peerConn         *webrtc.PeerConnection
	dc               *ctxwebrtc.DataChannel
	wonLastBattle    bool
	randSource       rand.Source

	battleMu     sync.Mutex
	battleNumber int
	battle       *Battle

	abortedMu sync.Mutex
	aborted   bool

	remoteInitCh chan []byte
}

func (m *Match) Battle() *Battle {
	m.battleMu.Lock()
	defer m.battleMu.Unlock()
	return m.battle
}

func (m *Match) Abort() {
	m.abortedMu.Lock()
	defer m.abortedMu.Unlock()
	m.aborted = true
}

func (m *Match) Aborted() bool {
	m.abortedMu.Lock()
	defer m.abortedMu.Unlock()
	return m.aborted
}

func New(conf config.Config, sessionID string, matchType uint16, gameTitle string, gameCRC32 uint32) *Match {
	return &Match{
		conf:      conf,
		sessionID: sessionID,
		matchType: matchType,
		gameTitle: gameTitle,
		gameCRC32: gameCRC32,

		negotiationErrCh: make(chan error),

		remoteInitCh: make(chan []byte),
	}
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

	log.Printf("local SDP: %s", peerConn.LocalDescription().SDP)
	log.Printf("remote SDP: %s", peerConn.RemoteDescription().SDP)

	var nonce [16]byte
	if _, err := cryptorand.Read(nonce[:]); err != nil {
		return fmt.Errorf("failed to generate rng seed part: %w", err)
	}

	log.Printf("our rng seed part: %s", hex.EncodeToString(nonce[:]))

	commitment := syncrand.Commit(nonce[:])
	var helloPacket packets.Hello
	helloPacket.ProtocolVersion = packets.ProtocolVersion
	copy(helloPacket.GameTitle[:], []byte(m.gameTitle))
	helloPacket.GameCRC32 = m.gameCRC32
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
		return ErrProtocolVersionMismatch
	}

	if theirHello.MatchType != m.matchType {
		return ErrMatchTypeMismatch
	}

	// MEGAMAN6 or ROCKEXE6 must match.
	if string(theirHello.GameTitle[:8]) != m.gameTitle[:8] {
		return ErrGameTypeMismatch
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

	log.Printf("their rng seed part: %s", hex.EncodeToString(theirNonce[:]))

	seed := syncrand.MakeSeed(nonce[:], theirNonce[:])
	log.Printf("rng seed: %s", hex.EncodeToString(seed))

	randSource := syncrand.NewSource(seed)

	m.peerConn = peerConn
	m.dc = dc
	m.randSource = randSource
	rng := rand.New(m.randSource)
	m.wonLastBattle = (rng.Int31n(2) == 1) == (connectionSide == signorclient.ConnectionSideOfferer)
	log.Printf("negotiation complete!")
	return nil
}

func (m *Match) handleConn(ctx context.Context) error {
	for {
		packet, trailer, err := packets.Recv(ctx, m.dc)
		if err != nil {
			return err
		}

		switch p := packet.(type) {
		case packets.Init:
			if p.BattleNumber != uint8(m.battleNumber) {
				log.Fatalf("mismatched battle number, expected %d but got %d", m.battleNumber, p.BattleNumber)
			}
			select {
			case m.remoteInitCh <- p.Marshaled[:]:
			case <-ctx.Done():
				return ctx.Err()
			}
		case packets.Input:
			battle := m.Battle()
			if battle == nil {
				log.Printf("no battle in progress, dropping input")
				continue
			}
			if p.BattleNumber != uint8(battle.number) {
				log.Printf("mismatched battle number, expected %d but got %d, dropping input", battle.number, p.BattleNumber)
				continue
			}
			battle.AddInput(ctx, m.battle.RemotePlayerIndex(), input.Input{LocalTick: int(p.LocalTick), RemoteTick: int(p.RemoteTick), Joyflags: p.Joyflags, CustomScreenState: p.CustomScreenState, Turn: trailer})
		}
	}
}

func (m *Match) EndBattle() error {
	m.battleMu.Lock()
	defer m.battleMu.Unlock()
	return m.endBattleLocked()
}

func (m *Match) endBattleLocked() error {
	log.Printf("battle ended, won = %t", m.wonLastBattle)

	if err := m.battle.Close(); err != nil {
		return err
	}
	m.battle = nil
	m.battleNumber++
	return nil
}

func (m *Match) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	defer m.Close()

	if err := m.negotiate(ctx); err != nil {
		m.negotiationErrCh <- err
		return err
	}
	close(m.negotiationErrCh)

	return m.handleConn(ctx)
}

func (m *Match) Close() error {
	m.battleMu.Lock()
	defer m.battleMu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.battle != nil {
		m.endBattleLocked()
	}
	if m.dc != nil {
		if err := m.dc.Close(); err != nil {
			return err
		}
		m.dc = nil
	}
	if m.peerConn != nil {
		if err := m.peerConn.Close(); err != nil {
			return err
		}
		m.peerConn = nil
	}
	return nil
}

func (m *Match) PollForReady(ctx context.Context) error {
	select {
	case err := <-m.negotiationErrCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrNotReady
	}
}

func (m *Match) SendInit(ctx context.Context, init []byte) error {
	var pkt packets.Init
	pkt.BattleNumber = uint8(m.Battle().number)
	copy(pkt.Marshaled[:], init)
	return packets.Send(ctx, m.dc, pkt, nil)
}

func (m *Match) SendInput(ctx context.Context, localTick uint32, remoteTick uint32, joyflags uint16, customScreenState uint8, turn []byte) error {
	var pkt packets.Input
	pkt.BattleNumber = uint8(m.Battle().number)
	pkt.LocalTick = localTick
	pkt.RemoteTick = remoteTick
	pkt.Joyflags = joyflags
	pkt.CustomScreenState = customScreenState
	return packets.Send(ctx, m.dc, pkt, turn)
}

func (m *Match) SetWonLastBattle(v bool) {
	m.wonLastBattle = v
}

func (m *Match) ReadRemoteInit(ctx context.Context) ([]byte, error) {
	select {
	case init := <-m.remoteInitCh:
		return init, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *Match) RandSource() rand.Source {
	return m.randSource
}

func (m *Match) Type() uint16 {
	return m.matchType
}
