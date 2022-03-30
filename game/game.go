package game

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/murkland/bbn6/av"
	"github.com/murkland/bbn6/bn6"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/input"
	"github.com/murkland/bbn6/match"
	"github.com/murkland/bbn6/mgba"
	"github.com/murkland/bbn6/trapper"
	"github.com/ncruces/zenity"
	"golang.org/x/sync/errgroup"
)

type Game struct {
	conf config.Config

	mainCore      *mgba.Core
	fastforwarder *fastforwarder

	pressedKeys []ebiten.Key

	bn6 *bn6.BN6

	vb *av.VideoBuffer

	fbuf   *image.RGBA
	fbufMu sync.Mutex

	audioCtx        *audio.Context
	gameAudioPlayer *audio.Player

	t *mgba.Thread

	match   *match.Match
	matchMu sync.Mutex

	debugSpew bool
}

func New(conf config.Config, romPath string) (*Game, error) {
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
	ebiten.SetWindowTitle("bbn6: " + mainCore.GameTitle())

	fastforwarder, err := newFastforwarder(romPath, bn6)
	if err != nil {
		return nil, err
	}

	audioCtx := audio.NewContext(mainCore.Options().SampleRate)

	width, height := mainCore.DesiredVideoDimensions()
	vb := av.NewVideoBuffer(width, height)
	ebiten.SetWindowSize(width*3, height*3)

	mainCore.SetVideoBuffer(vb.Pointer(), width)
	t := mgba.NewThread(mainCore)
	if !t.Start() {
		return nil, err
	}
	mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))

	gameAudioPlayer, err := audioCtx.NewPlayer(av.NewAudioReader(mainCore, mainCore.Options().SampleRate))
	if err != nil {
		return nil, err
	}
	gameAudioPlayer.SetBufferSize(time.Duration(mainCore.Options().AudioBuffers+0x4) * time.Second / time.Duration(mainCore.Options().SampleRate))
	gameAudioPlayer.Play()

	g := &Game{
		conf: conf,

		mainCore:      mainCore,
		fastforwarder: fastforwarder,

		bn6: bn6,

		vb: vb,

		audioCtx:        audioCtx,
		gameAudioPlayer: gameAudioPlayer,

		t: t,
	}
	g.InstallTraps(mainCore)

	return g, nil
}

func (g *Game) RunBackgroundTasks(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() error {
		g.serviceFbuf()
		return nil
	})

	return errg.Wait()
}

func (g *Game) serviceFbuf() {
	runtime.LockOSThread()
	for {
		if g.mainCore.GBA().Sync().WaitFrameStart() {
			g.fbufMu.Lock()
			g.fbuf = g.vb.CopyImage()
			g.fbufMu.Unlock()
		} else {
			// TODO: Optimize this.
			time.Sleep(500 * time.Microsecond)
		}
		g.mainCore.GBA().Sync().WaitFrameEnd()
	}
}

func (g *Game) InstallTraps(core *mgba.Core) error {
	tp := trapper.New(core)

	tp.Add(g.bn6.Offsets.ROM.A_battle_init__call__battle_copyInputData, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("attempting to copy input data while no battle was active!")
		}

		core.GBA().SetRegister(0, 0x0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_init_marshal__ret, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("attempting to marshal init data while no battle was active!")
		}

		ctx := context.Background()

		localInit := g.bn6.LocalMarshaledBattleState(core)
		if err := match.SendInit(ctx, localInit); err != nil {
			log.Fatalf("failed to send init info: %s", err)
		}

		log.Printf("init sent")
		g.bn6.SetPlayerMarshaledBattleState(core, battle.LocalPlayerIndex(), localInit)

		remoteInit, err := battle.ReadRemoteInit(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				g.endMatch()
				g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
				// TODO: Figure out how to gracefully exit the battle.
				log.Fatalf("TODO: drop the connection")
			}
			log.Fatalf("failed to receive init info: %s", err)
		}

		log.Printf("init received")
		g.bn6.SetPlayerMarshaledBattleState(core, battle.RemotePlayerIndex(), remoteInit)
		committedState := core.SaveState()
		battle.SetCommittedState(committedState)

		if err := battle.ReplayWriter().WriteState(battle.LocalPlayerIndex(), committedState); err != nil {
			log.Fatalf("failed to write to replay: %s", err)
		}
		if err := battle.ReplayWriter().WriteInit(battle.LocalPlayerIndex(), localInit); err != nil {
			log.Fatalf("failed to write to replay: %s", err)
		}
		if err := battle.ReplayWriter().WriteInit(battle.RemotePlayerIndex(), remoteInit); err != nil {
			log.Fatalf("failed to write to replay: %s", err)
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_turn_marshal__ret, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("attempting to marshal turn data while no battle was active!")
		}

		battle.AddLocalPendingTurn(g.bn6.LocalMarshaledBattleState(core))
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_update__call__battle_copyInputData, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("attempting to copy input data while no battle was active!")
		}

		core.GBA().SetRegister(0, 0x0)
		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()

		ctx := context.Background()

		tick := g.bn6.InBattleTime(core)

		nextJoyflags := g.bn6.LocalJoyflags(core)
		joyflags := battle.AddLocalBufferedInputAndConsume(nextJoyflags)

		customScreenState := g.bn6.LocalCustomScreenState(core)

		turn := battle.ConsumeLocalPendingTurn()

		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := battle.AddInput(ctx, battle.LocalPlayerIndex(), input.Input{Tick: int(tick), Joyflags: joyflags, CustomScreenState: customScreenState, Turn: turn}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				log.Printf("could not queue local input within %s, dropping connection", timeout)
				g.endMatch()
				g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
				// TODO: Figure out how to gracefully exit the battle.
				log.Fatalf("TODO: drop the connection")
			}
			log.Fatalf("failed to add input: %s", err)
		}

		if err := match.SendInput(ctx, tick, joyflags, customScreenState, turn); err != nil {
			log.Fatalf("failed to send input: %s", err)
		}

		inputPairs, left := battle.ConsumeInputs()
		committedState, dirtyState, err := g.fastforwarder.fastforward(battle.CommittedState(), battle.ReplayWriter(), battle.LocalPlayerIndex(), inputPairs, battle.LastCommittedRemoteInput(), left)
		if err != nil {
			log.Fatalf("failed to fastforward: %s", err)
		}
		battle.SetCommittedState(committedState)
		g.mainCore.LoadState(dirtyState)
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_runUnpausedStep__cmp__retval, func() {
		match := g.Match()
		if match == nil {
			return
		}

		r := core.GBA().Register(0)
		if r == 1 {
			match.SetWonLastBattle(true)
		} else if r == 2 {
			match.SetWonLastBattle(false)
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_updating__ret__go_to_custom_screen, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("turn ended while no battle was active!")
		}

		tick := g.bn6.InBattleTime(core)
		log.Printf("turn ended on %d, rng state = %08x", tick, g.bn6.RNG2State(core))
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_start__ret, func() {
		match := g.Match()
		if match == nil {
			return
		}

		if err := match.NewBattle(g.mainCore); err != nil {
			log.Fatalf("failed to start new battle: %s", err)
		}
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_end__entry, func() {
		match := g.Match()
		if match == nil {
			return
		}

		if err := match.EndBattle(); err != nil {
			log.Fatalf("failed to end battle: %s", err)
		}

		g.mainCore.GBA().Sync().SetFPSTarget(float32(expectedFPS))
	})

	tp.Add(g.bn6.Offsets.ROM.A_battle_isP2__tst, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("attempted to get battle p2 information while no battle was active!")
		}

		core.GBA().SetRegister(0, uint32(battle.LocalPlayerIndex()))
	})

	tp.Add(g.bn6.Offsets.ROM.A_link_isP2__ret, func() {
		match := g.Match()
		if match == nil {
			return
		}

		battle := match.Battle()
		if battle == nil {
			log.Fatalf("attempted to get link p2 information while no battle was active!")
		}

		core.GBA().SetRegister(0, uint32(battle.LocalPlayerIndex()))
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_handleLinkCableInput__entry, func() {
		log.Printf("unhandled call to commMenu_handleLinkCableInput at 0x%08x: uh oh!", core.GBA().Register(15)-4)
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput, func() {
		ctx := context.Background()

		g.matchMu.Lock()
		defer g.matchMu.Unlock()

		if g.match == nil {
			volume := g.gameAudioPlayer.Volume()
			g.gameAudioPlayer.SetVolume(0)
			code, err := zenity.Entry("Enter a code to matchmake with:", zenity.Title("bbn6"))
			g.gameAudioPlayer.SetVolume(volume)
			if err != nil {
				log.Printf("matchmaking dialog did not return a code: %s", err)
				g.bn6.DropMatchmakingFromCommMenu(core)
			} else {
				match, err := match.New(g.conf, code, g.bn6.BattleType(g.mainCore), g.mainCore.GameTitle())
				if err != nil {
					// TODO: handle this better.
					log.Fatalf("failed to start match: %s", err)
				}
				g.match = match
				// TODO: Check the errors from this.
				go g.match.Run(ctx)
			}
		}

		if g.match != nil {
			ready, err := g.match.PollForReady(ctx)
			if err != nil {
				log.Fatalf("match did not become ready: %s", err)
			}

			if ready {
				g.bn6.StartBattleFromCommMenu(core)
				log.Printf("match started")
			}
		}

		core.GBA().SetRegister(15, core.GBA().Register(15)+0x4)
		core.GBA().ThumbWritePC()
	})

	tp.Add(g.bn6.Offsets.ROM.A_commMenu_initBattle__entry, func() {
		match := g.Match()
		if match == nil {
			return
		}
		battleSettingsAndBackground := match.RandomBattleSettingsAndBackground()
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

	core.InstallBeefTrap(tp.BeefHandler)

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

	if err := (func() error {
		match := g.Match()
		if match != nil {
			battle := match.Battle()
			if battle != nil {
				expected := match.RunaheadTicksAllowed()
				lag := battle.Lag()
				tps := expectedFPS - (lag - expected)
				// TODO: Not thread safe.
				g.mainCore.GBA().Sync().SetFPSTarget(float32(tps))
			}
		}

		return nil
	})(); err != nil {
		return err
	}

	g.pressedKeys = inpututil.AppendPressedKeys(g.pressedKeys[:0])

	var keys mgba.Keys
	for _, key := range g.pressedKeys {
		if key == g.conf.Keymapping.A {
			keys |= mgba.KeysA
		}
		if key == g.conf.Keymapping.B {
			keys |= mgba.KeysB
		}
		if key == g.conf.Keymapping.L {
			keys |= mgba.KeysL
		}
		if key == g.conf.Keymapping.R {
			keys |= mgba.KeysR
		}
		if key == g.conf.Keymapping.Left {
			keys |= mgba.KeysLeft
		}
		if key == g.conf.Keymapping.Right {
			keys |= mgba.KeysRight
		}
		if key == g.conf.Keymapping.Up {
			keys |= mgba.KeysUp
		}
		if key == g.conf.Keymapping.Down {
			keys |= mgba.KeysDown
		}
		if key == g.conf.Keymapping.Start {
			keys |= mgba.KeysStart
		}
		if key == g.conf.Keymapping.Select {
			keys |= mgba.KeysSelect
		}
	}
	g.mainCore.SetKeys(keys)

	if g.conf.Keymapping.DebugSpew != -1 && inpututil.IsKeyJustPressed(g.conf.Keymapping.DebugSpew) {
		g.debugSpew = !g.debugSpew
	}

	return nil
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.mainCore.DesiredVideoDimensions()
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.fbufMu.Lock()
	defer g.fbufMu.Unlock()

	if g.fbuf == nil {
		return
	}

	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(ebiten.NewImageFromImage(g.fbuf), opts)

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
