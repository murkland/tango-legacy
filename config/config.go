package config

import (
	"io"

	"github.com/BurntSushi/toml"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pion/webrtc/v3"
)

type Keymapping struct {
	A         ebiten.Key
	B         ebiten.Key
	L         ebiten.Key
	R         ebiten.Key
	Left      ebiten.Key
	Right     ebiten.Key
	Up        ebiten.Key
	Down      ebiten.Key
	Start     ebiten.Key
	Select    ebiten.Key
	DebugSpew ebiten.Key
}

type Matchmaking struct {
	ConnectAddr string
}

type Config struct {
	Keymapping  Keymapping
	Matchmaking Matchmaking
	WebRTC      webrtc.Configuration
}

var DefaultConfig = Config{
	Keymapping: Keymapping{
		A:         ebiten.KeyZ,
		B:         ebiten.KeyX,
		L:         ebiten.KeyA,
		R:         ebiten.KeyS,
		Left:      ebiten.KeyArrowLeft,
		Right:     ebiten.KeyArrowRight,
		Up:        ebiten.KeyArrowUp,
		Down:      ebiten.KeyArrowDown,
		Start:     ebiten.KeyEnter,
		Select:    ebiten.KeyBackspace,
		DebugSpew: ebiten.KeyBackquote,
	},
	Matchmaking: Matchmaking{
		ConnectAddr: "bbn6.murk.land:12345",
	},
	WebRTC: webrtc.Configuration{
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
	},
}

type RawKeymapping struct {
	A         string
	B         string
	L         string
	R         string
	Left      string
	Right     string
	Up        string
	Down      string
	Start     string
	Select    string
	DebugSpew string
}

type RawConfig struct {
	Keymapping  RawKeymapping
	Matchmaking Matchmaking
	WebRTC      webrtc.Configuration
}

func (c Config) ToRaw() RawConfig {
	return RawConfig{
		Keymapping: RawKeymapping{
			A:         keyName(c.Keymapping.A),
			B:         keyName(c.Keymapping.B),
			L:         keyName(c.Keymapping.L),
			R:         keyName(c.Keymapping.R),
			Left:      keyName(c.Keymapping.Left),
			Right:     keyName(c.Keymapping.Right),
			Up:        keyName(c.Keymapping.Up),
			Down:      keyName(c.Keymapping.Down),
			Start:     keyName(c.Keymapping.Start),
			Select:    keyName(c.Keymapping.Select),
			DebugSpew: keyName(c.Keymapping.DebugSpew),
		},
		Matchmaking: c.Matchmaking,
		WebRTC:      c.WebRTC,
	}
}

func (rc RawConfig) ToParsed() Config {
	return Config{
		Keymapping: Keymapping{
			A:         keyCode(rc.Keymapping.A),
			B:         keyCode(rc.Keymapping.B),
			L:         keyCode(rc.Keymapping.L),
			R:         keyCode(rc.Keymapping.R),
			Left:      keyCode(rc.Keymapping.Left),
			Right:     keyCode(rc.Keymapping.Right),
			Up:        keyCode(rc.Keymapping.Up),
			Down:      keyCode(rc.Keymapping.Down),
			Start:     keyCode(rc.Keymapping.Start),
			Select:    keyCode(rc.Keymapping.Select),
			DebugSpew: keyCode(rc.Keymapping.DebugSpew),
		},
		Matchmaking: rc.Matchmaking,
		WebRTC:      rc.WebRTC,
	}
}
func Save(config Config, w io.Writer) error {
	return toml.NewEncoder(w).Encode(config.ToRaw())
}

func Load(r io.Reader) (Config, error) {
	var rawConfig RawConfig

	if _, err := toml.NewDecoder(r).Decode(&rawConfig); err != nil {
		return Config{}, err
	}

	return rawConfig.ToParsed(), nil
}
