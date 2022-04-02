package match

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/keegancsmith/nth"
	"github.com/murkland/clone"
	"github.com/murkland/ctxwebrtc"
	"github.com/murkland/ringbuf"
	signorclient "github.com/murkland/signor/client"
	"github.com/murkland/syncrand"
	"github.com/murkland/tango/config"
	"github.com/murkland/tango/input"
	"github.com/murkland/tango/mgba"
	"github.com/murkland/tango/packets"
	"github.com/murkland/tango/replay"
	"github.com/pion/webrtc/v3"
	"golang.org/x/exp/constraints"
	"golang.org/x/sync/errgroup"
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

	delayRingbuf   *ringbuf.RingBuf[time.Duration]
	delayRingbufMu sync.RWMutex

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

func New(conf config.Config, sessionID string, matchType uint16, gameTitle string, gameCRC32 uint32) (*Match, error) {
	return &Match{
		conf:      conf,
		sessionID: sessionID,
		matchType: matchType,
		gameTitle: gameTitle,
		gameCRC32: gameCRC32,

		negotiationErrCh: make(chan error),

		delayRingbuf: ringbuf.New[time.Duration](9),

		remoteInitCh: make(chan []byte),
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

type orderableSlice[T constraints.Ordered] []T

func (s orderableSlice[T]) Len() int {
	return len(s)
}

func (s orderableSlice[T]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s orderableSlice[T]) Less(i, j int) bool {
	return s[i] < s[j]
}

func (m *Match) MedianDelay() time.Duration {
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
			select {
			case m.remoteInitCh <- p.Marshaled[:]:
			case <-ctx.Done():
				return ctx.Err()
			}
		case packets.Input:
			battle := m.Battle()
			if battle == nil {
				log.Printf("received input packet while battle was apparently not active, dropping it (this may cause a desync!)")
				continue
			}
			battle.AddInput(ctx, m.battle.RemotePlayerIndex(), input.Input{Tick: int(p.ForTick), Joyflags: p.Joyflags, CustomScreenState: p.CustomScreenState, Turn: trailer})
		}
	}
}

const localInputBufferSize = 2

func (m *Match) NewBattle(core *mgba.Core) error {
	m.battleMu.Lock()
	defer m.battleMu.Unlock()

	if m.battle != nil {
		return errors.New("battle already started")
	}

	b := &Battle{
		isP2: !m.wonLastBattle,

		lastCommittedRemoteInput: input.Input{Joyflags: 0xfc00},

		localInputBuffer: ringbuf.New[uint16](localInputBufferSize),

		iq: input.NewQueue(60),
	}

	fn := filepath.Join("replays", fmt.Sprintf("%s_p%d.tangoreplay", time.Now().Format("20060102030405"), b.LocalPlayerIndex()+1))
	log.Printf("writing replay: %s", fn)

	il, err := replay.NewWriter(fn, core)
	if err != nil {
		return err
	}
	b.rw = il
	m.battle = b
	m.battleNumber++
	log.Printf("battle %d started, won last battle (is p1) = %t", m.battleNumber, m.wonLastBattle)
	return nil
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

func (m *Match) RunaheadTicksAllowed() int {
	expected := int(m.MedianDelay()*time.Duration(expectedFPS)/2/time.Second + 1)
	if expected < 1 {
		expected = 1
	}
	return expected
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
	copy(pkt.Marshaled[:], init)
	return packets.Send(ctx, m.dc, pkt, nil)
}

func (m *Match) SendInput(ctx context.Context, tick uint32, joyflags uint16, customScreenState uint8, turn []byte) error {
	var pkt packets.Input
	pkt.ForTick = uint32(tick)
	pkt.Joyflags = joyflags
	pkt.CustomScreenState = customScreenState
	return packets.Send(ctx, m.dc, pkt, turn)
}

func (m *Match) SetWonLastBattle(v bool) {
	m.wonLastBattle = v
}

var battleBackgrounds = []uint16{
	0x00, 0x01, 0x01, 0x03, 0x04, 0x05, 0x06,
	0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
	0x0e, 0x0f, 0x10, 0x11, 0x11, 0x13, 0x13,
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
