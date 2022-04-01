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

func (k Keymapping) ToRaw() RawKeymapping {
	return RawKeymapping{
		A:         keyName(k.A),
		B:         keyName(k.B),
		L:         keyName(k.L),
		R:         keyName(k.R),
		Left:      keyName(k.Left),
		Right:     keyName(k.Right),
		Up:        keyName(k.Up),
		Down:      keyName(k.Down),
		Start:     keyName(k.Start),
		Select:    keyName(k.Select),
		DebugSpew: keyName(k.DebugSpew),
	}
}

type Matchmaking struct {
	ConnectAddr string
}

type AudioInterpolationType int

const (
	AudioInterpolationTypeRubbery AudioInterpolationType = iota
	AudioInterpolationTypeClippy
)

type Audio struct {
	Interpolation AudioInterpolationType
}

func (a Audio) ToRaw() RawAudio {
	interpolation := "clippy"
	switch a.Interpolation {
	case AudioInterpolationTypeRubbery:
		interpolation = "rubbery"
	case AudioInterpolationTypeClippy:
		interpolation = "clippy"
	}
	return RawAudio{
		Interpolation: interpolation,
	}
}

type Config struct {
	Keymapping  Keymapping
	Audio       Audio
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
		ConnectAddr: "bbn6mm.murk.land:80",
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

func (rk RawKeymapping) ToParsed() Keymapping {
	return Keymapping{
		A:         keyCode(rk.A),
		B:         keyCode(rk.B),
		L:         keyCode(rk.L),
		R:         keyCode(rk.R),
		Left:      keyCode(rk.Left),
		Right:     keyCode(rk.Right),
		Up:        keyCode(rk.Up),
		Down:      keyCode(rk.Down),
		Start:     keyCode(rk.Start),
		Select:    keyCode(rk.Select),
		DebugSpew: keyCode(rk.DebugSpew),
	}
}

type RawAudio struct {
	Interpolation string
}

type RawConfig struct {
	Keymapping  RawKeymapping
	Audio       RawAudio
	Matchmaking Matchmaking
	WebRTC      webrtc.Configuration
}

func (c Config) ToRaw() RawConfig {
	return RawConfig{
		Keymapping:  c.Keymapping.ToRaw(),
		Audio:       c.Audio.ToRaw(),
		Matchmaking: c.Matchmaking,
		WebRTC:      c.WebRTC,
	}
}

func (rc RawConfig) ToParsed() Config {
	return Config{
		Keymapping:  rc.Keymapping.ToParsed(),
		Audio:       rc.Audio.ToParsed(),
		Matchmaking: rc.Matchmaking,
		WebRTC:      rc.WebRTC,
	}
}

func (ra RawAudio) ToParsed() Audio {
	interpolation := AudioInterpolationTypeClippy
	switch ra.Interpolation {
	case "rubbery":
		interpolation = AudioInterpolationTypeRubbery
	case "clippy":
		interpolation = AudioInterpolationTypeClippy
	}
	return Audio{
		Interpolation: interpolation,
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
