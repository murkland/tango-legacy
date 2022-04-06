package game

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/murkland/tango/av"
	"github.com/murkland/tango/bn6"
	"github.com/murkland/tango/config"
	"github.com/murkland/tango/input"
	"github.com/murkland/tango/match"
	"github.com/murkland/tango/mgba"
	"github.com/ncruces/zenity"
	"golang.org/x/text/message"
)

type Game struct {
	conf config.Config
	p    *message.Printer

	mainCore      *mgba.Core
	fastforwarder *Fastforwarder

	joyflags mgba.Keys

	bn6 *bn6.BN6

	vb      *av.VideoBuffer
	vbPixMu sync.Mutex
	vbPix   []byte

	fbuf *ebiten.Image

	audioCtx        *audio.Context
	gameAudioPlayer *audio.Player

	t *mgba.Thread

	match   *match.Match
	matchMu sync.Mutex

	debugSpew bool
}

func New(conf config.Config, p *message.Printer, romPath string) (*Game, error) {
	mainCore, err := newCore(romPath)
	if err != nil {
		return nil, err
	}

	romFilename := filepath.Base(romPath)
	ext := filepath.Ext(romFilename)
	savePath := filepath.Join("saves", romFilename[:len(romFilename)-len(ext)]+".sav")
	saveVF := mgba.OpenVF(savePath, os.O_CREATE|os.O_RDWR)
	if saveVF == nil {
		return nil, errors.New("failed to open save file")
	}

	if err := mainCore.LoadSave(saveVF); err != nil {
		return nil, err
	}
	log.Printf("loaded save file: %s", savePath)

	bn6 := bn6.Load(mainCore.GameTitle())
	if bn6 == nil {
		return nil, fmt.Errorf("unsupported game: %s", mainCore.GameTitle())
	}
	ebiten.SetWindowTitle("tango: " + mainCore.GameTitle())

	fastforwarder, err := NewFastforwarder(romPath, bn6)
	if err != nil {
		return nil, err
	}

	audioCtx := audio.NewContext(mainCore.Options().SampleRate)

	width, height := mainCore.DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)
	mainCore.SetVideoBuffer(vb.Pointer(), width)
	ebiten.SetWindowSize(width*3, height*3)

	newAudioReader := av.NewClippyAudioReader
	switch conf.Audio.Interpolation {
	case config.AudioInterpolationTypeClippy:
		newAudioReader = av.NewClippyAudioReader
	case config.AudioInterpolationTypeRubbery:
		newAudioReader = av.NewRubberyAudioReader
	}

	gameAudioPlayer, err := audioCtx.NewPlayer(newAudioReader(mainCore, mainCore.Options().SampleRate))
	if err != nil {
		return nil, err
	}
	gameAudioPlayer.SetBufferSize(time.Duration(mainCore.AudioBufferSize()+1) * time.Second / time.Duration(mainCore.Options().SampleRate))

	g := &Game{
		conf: conf,
		p:    p,

		mainCore:      mainCore,
		fastforwarder: fastforwarder,

		bn6: bn6,

		vb:    vb,
		vbPix: make([]byte, width*height*4),

		fbuf: ebiten.NewImage(width, height),

		audioCtx:        audioCtx,
		gameAudioPlayer: gameAudioPlayer,
	}
	g.InstallTraps(mainCore)

	g.t = mgba.NewThread(mainCore)
	g.t.SetFrameCallback(func() {
		g.vbPixMu.Lock()
		defer g.vbPixMu.Unlock()
		copy(g.vbPix, g.vb.Pix())
	})

	if !g.t.Start() {
		return nil, err
	}
	mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))

	gameAudioPlayer.Play()

	return g, nil
}

func (g *Game) InstallTraps(core *mgba.Core) error {
	tp := mgba.NewTrapper(core)

	tp.Add(g.bn6.Offsets.ROM.A_battle_init__call__battle_copyInputData, func() {
		m := g.Match()
		if m == nil {
			return
		}

		battle := m.Battle()
		if battle == nil {
			log.Panicf("attempting to copy init data while no battle was active!")
		}

		core.GBA().SetRegister(0, 0x0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_init_marshal__ret, func() {
		m := g.Match()
		if m == nil {
			return
		}

		battle := m.Battle()
		if battle == nil {
			log.Panicf("attempting to marshal init data while no battle was active!")
		}

		ctx := context.Background()

		localInit := g.bn6.LocalMarshaledBattleState(core)
		if err := m.SendInit(ctx, localInit); err != nil {
			log.Panicf("failed to send init info: %s", err)
		}

		log.Printf("init sent")
		g.bn6.SetPlayerMarshaledBattleState(core, battle.LocalPlayerIndex(), localInit)

		remoteInit, err := m.ReadRemoteInit(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
				m.Abort()
				return
			}
			log.Panicf("failed to receive init info: %s", err)
		}

		log.Printf("init received")
		g.bn6.SetPlayerMarshaledBattleState(core, battle.RemotePlayerIndex(), remoteInit)

		if err := battle.ReplayWriter().WriteInit(battle.LocalPlayerIndex(), localInit); err != nil {
			log.Panicf("failed to write to replay: %s", err)
		}
		if err := battle.ReplayWriter().WriteInit(battle.RemotePlayerIndex(), remoteInit); err != nil {
			log.Panicf("failed to write to replay: %s", err)
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_turn_marshal__ret, func() {
		m := g.Match()
		if m == nil {
			return
		}

		battle := m.Battle()
		if battle == nil {
			log.Panicf("attempting to marshal turn data while no battle was active!")
		}

		log.Printf("turn data marshaled on %d", g.bn6.InBattleTime(g.mainCore))
		battle.AddLocalPendingTurn(g.bn6.LocalMarshaledBattleState(core))
	})

	tp.Add(g.bn6.Offsets.ROM.A_main__readJoyflags, func() {
		m := g.Match()
		if m == nil {
			return
		}

		if m.Aborted() {
			return
		}

		battle := m.Battle()
		if battle == nil {
			return
		}

		if !battle.IsAcceptingInput() {
			return
		}

		ctx := context.Background()

		inBattleTime := int(g.bn6.InBattleTime(g.mainCore))

		if battle.CommittedState() == nil {
			committedState := core.SaveState()
			battle.SetCommittedState(committedState)

			log.Printf("battle state committed")

			if err := battle.ReplayWriter().WriteState(battle.LocalPlayerIndex(), committedState); err != nil {
				log.Panicf("failed to write to replay: %s", err)
			}
		}

		joyflags := uint16(g.joyflags | 0xfc00)
		localTick := inBattleTime
		lastCommittedRemoteInput := battle.LastCommittedRemoteInput()
		remoteTick := lastCommittedRemoteInput.LocalTick

		customScreenState := g.bn6.LocalCustomScreenState(core)

		turn := battle.ConsumeLocalPendingTurn()

		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := battle.AddInput(ctx, battle.LocalPlayerIndex(), input.Input{LocalTick: localTick, RemoteTick: remoteTick, Joyflags: joyflags, CustomScreenState: customScreenState, Turn: turn}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				log.Printf("could not queue local input within %s, dropping connection", timeout)
				g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
				m.Abort()
				return
			}
			log.Panicf("failed to add input: %s", err)
		}

		if err := m.SendInput(ctx, uint32(localTick), uint32(remoteTick), joyflags, customScreenState, turn); err != nil {
			log.Panicf("failed to send input: %s", err)
		}

		inputPairs, left := battle.ConsumeInputs()
		committedState, dirtyState, err := g.fastforwarder.Fastforward(battle.CommittedState(), battle.ReplayWriter(), battle.LocalPlayerIndex(), inputPairs, battle.LastCommittedRemoteInput(), left)
		if err != nil {
			log.Panicf("failed to fastforward: %s\n  inputPairs = %+v\n  left = %+v", err, inputPairs, left)
		}
		battle.SetCommittedState(committedState)

		tps := expectedFPS + (remoteTick - localTick) - (lastCommittedRemoteInput.RemoteTick - lastCommittedRemoteInput.LocalTick)
		g.mainCore.GBA().Sync().SetFPSTarget(float32(tps))

		// This will jump to after A_battle_update__call__battle_copyInputData.
		if !g.mainCore.LoadState(dirtyState) {
			log.Panicf("failed to load dirty state")
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		m := g.Match()
		if m == nil {
			return
		}

		core.GBA().SetRegister(0, 0x0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()

		battle := m.Battle()
		if battle == nil {
			return
		}

		if !battle.IsAcceptingInput() {
			battle.StartAcceptingInput()
			return
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_runUnpausedStep__cmp__retval, func() {
		m := g.Match()
		if m == nil {
			return
		}

		battle := m.Battle()
		if battle == nil {
			return
		}

		switch core.GBA().Register(0) {
		case 1:
			m.SetWonLastBattle(true)
		case 2:
			m.SetWonLastBattle(false)
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_start__ret, func() {
		m := g.Match()
		if m == nil {
			return
		}

		if err := m.NewBattle(g.mainCore); err != nil {
			log.Panicf("failed to start new battle: %s", err)
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_ending__ret, func() {
		m := g.Match()
		if m == nil {
			return
		}

		if err := m.EndBattle(); err != nil {
			log.Panicf("failed to end battle: %s", err)
		}

		g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		m := g.Match()
		if m == nil {
			return
		}

		battle := m.Battle()
		if battle == nil {
			log.Panicf("attempted to get battle p2 information while no battle was active!")
		}

		core.GBA().SetRegister(0, uint32(battle.LocalPlayerIndex()))
	})

	tp.Add(g.bn6.Offsets.ROM.A_link_isP2__ret, func() {
		m := g.Match()
		if m == nil {
			return
		}

		battle := m.Battle()
		if battle == nil {
			log.Panicf("attempted to get link p2 information while no battle was active!")
		}

		core.GBA().SetRegister(0, uint32(battle.LocalPlayerIndex()))
	})

	tp.Add(g.bn6.Offsets.ROM.A_getCopyDataInputState__ret, func() {
		m := g.Match()
		if m == nil {
			return
		}

		r0 := core.GBA().Register(0)
		if r0 != 2 {
			log.Printf("expected getCopyDataInputState to be 2 but got %d", r0)
		}

		r0 = 2
		if m.Aborted() {
			r0 = 4
		}
		core.GBA().SetRegister(0, r0)
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_handleLinkCableInput__entry, func() {
		log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", core.GBA().Register(15)-4)
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()

		ctx := context.Background()

		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			volume := g.gameAudioPlayer.Volume()
			g.gameAudioPlayer.SetVolume(0)
			code, err := zenity.Entry(g.p.Sprintf("ENTER_MATCHMAKING_CODE"), zenity.Title("tango"))
			code = strings.ReplaceAll(strings.ToLower(code), " ", "")
			g.gameAudioPlayer.SetVolume(volume)
			if err != nil || code == "" {
				log.Printf("matchmaking dialog did not return a code: %s", err)
				g.bn6.DropMatchmakingFromCommMenu(core, 0)
			} else {
				match := match.New(g.conf, code, g.bn6.MatchType(g.mainCore), g.mainCore.GameTitle(), g.mainCore.CRC32())
				g.match = match
				go func() {
					if err := match.Run(ctx); err != nil {
						log.Printf("match ended with error: %s", err)
						match.Abort()
					}
				}()
			}
		}

		if g.match != nil {
			err := g.match.PollForReady(ctx)
			if err != nil {
				if errors.Is(err, match.ErrNotReady) {
					return
				}
				if errors.Is(err, match.ErrProtocolVersionMismatch) || errors.Is(err, match.ErrGameTypeMismatch) || errors.Is(err, match.ErrMatchTypeMismatch) {
					g.bn6.DropMatchmakingFromCommMenu(core, bn6.DropMatchmakingTypeWrongMode)
					log.Printf("mismatch: %s", err)
				} else {
					g.bn6.DropMatchmakingFromCommMenu(core, bn6.DropMatchmakingTypeConnectionError)
					log.Printf("failed to poll match: %s", err)
				}
				g.match = nil
				return
			}

			g.bn6.StartBattleFromCommMenu(core)
			log.Printf("match started")
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_initBattle__entry, func() {
		m := g.Match()
		if m == nil {
			return
		}
		battleSettingsAndBackground := g.bn6.RandomBattleSettingsAndBackground(m.RandSource(), uint8(m.Type()&0xff))
		log.Printf("selected battle settings and background: %04x", battleSettingsAndBackground)
		g.bn6.SetLinkBattleSettingsAndBackground(g.mainCore, battleSettingsAndBackground)
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_waitForFriend__ret__cancel, func() {
		log.Printf("match canceled by user")
		g.endMatch()

		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_endBattle__entry, func() {
		log.Printf("match ended")
		g.endMatch()
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_inBattle__call__commMenu_handleLinkCableInput, func() {
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Attach(core.GBA())

	return nil
}

func (g *Game) Finish() {
	g.t.End()
	g.t.Join()
}

const expectedFPS = 60

func (g *Game) Update() error {
	if g.t.HasCrashed() {
		return errors.New("mgba thread crashed")
	}

	g.joyflags = ebitenToMgbaKeys(g.conf.Keymapping, inpututil.AppendPressedKeys(nil))

	if g.Match() == nil {
		// Use regular input handling outside of a match.
		g.mainCore.SetKeys(g.joyflags)
	}

	if g.conf.Keymapping.DebugSpew != -1 && inpututil.IsKeyJustPressed(ebiten.Key(g.conf.Keymapping.DebugSpew)) {
		g.debugSpew = !g.debugSpew
	}

	return nil
}

func (g *Game) scaleFactor(bounds image.Rectangle) int {
	w, h := g.mainCore.DesiredVideoDimensions()
	k := bounds.Dx() / w
	if s := bounds.Dy() / h; s < k {
		k = s
	}
	return k
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.vbPixMu.Lock()
	defer g.vbPixMu.Unlock()

	k := g.scaleFactor(screen.Bounds())
	opts := &ebiten.DrawImageOptions{}
	w, h := g.mainCore.DesiredVideoDimensions()
	opts.GeoM.Scale(float64(k), float64(k))
	opts.GeoM.Translate(float64((screen.Bounds().Dx()-w*k)/2), float64((screen.Bounds().Dy()-h*k)/2))
	g.fbuf.ReplacePixels(g.vbPix)
	screen.DrawImage(g.fbuf, opts)

	if g.debugSpew {
		g.spewDebug(screen)
	}
}

func (g *Game) Match() *match.Match {
	g.matchMu.Lock()
	defer g.matchMu.Unlock()
	return g.match
}

func (g *Game) endMatch() error {
	g.matchMu.Lock()
	defer g.matchMu.Unlock()
	if err := g.match.Close(); err != nil {
		return err
	}
	g.match = nil
	return nil
}
