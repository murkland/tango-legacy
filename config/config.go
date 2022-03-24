package config

import (
	"io"

	"github.com/BurntSushi/toml"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pion/webrtc/v3"
)

type Keymapping struct {
	A      ebiten.Key
	B      ebiten.Key
	L      ebiten.Key
	R      ebiten.Key
	Left   ebiten.Key
	Right  ebiten.Key
	Up     ebiten.Key
	Down   ebiten.Key
	Start  ebiten.Key
	Select ebiten.Key
}

type Config struct {
	Keymapping Keymapping
	WebRTC     webrtc.Configuration
}

var DefaultConfig = Config{
	Keymapping: Keymapping{
		A:      ebiten.KeyZ,
		B:      ebiten.KeyX,
		L:      ebiten.KeyL,
		R:      ebiten.KeyR,
		Left:   ebiten.KeyArrowLeft,
		Right:  ebiten.KeyArrowRight,
		Up:     ebiten.KeyArrowUp,
		Down:   ebiten.KeyArrowDown,
		Start:  ebiten.KeyEnter,
		Select: ebiten.KeyBackspace,
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
	A      string
	B      string
	L      string
	R      string
	Left   string
	Right  string
	Up     string
	Down   string
	Start  string
	Select string
}

type RawConfig struct {
	Keymapping RawKeymapping
	WebRTC     webrtc.Configuration
}

func (c Config) ToRaw() RawConfig {
	return RawConfig{
		Keymapping: RawKeymapping{
			A:      keyCodeToKeyName[c.Keymapping.A],
			B:      keyCodeToKeyName[c.Keymapping.B],
			L:      keyCodeToKeyName[c.Keymapping.L],
			R:      keyCodeToKeyName[c.Keymapping.R],
			Left:   keyCodeToKeyName[c.Keymapping.Left],
			Right:  keyCodeToKeyName[c.Keymapping.Right],
			Up:     keyCodeToKeyName[c.Keymapping.Up],
			Down:   keyCodeToKeyName[c.Keymapping.Down],
			Start:  keyCodeToKeyName[c.Keymapping.Start],
			Select: keyCodeToKeyName[c.Keymapping.Select],
		},
		WebRTC: c.WebRTC,
	}
}

func (rc RawConfig) ToParsed() Config {
	return Config{
		Keymapping: Keymapping{
			A:      keyNameToKeyCode[rc.Keymapping.A],
			B:      keyNameToKeyCode[rc.Keymapping.B],
			L:      keyNameToKeyCode[rc.Keymapping.L],
			R:      keyNameToKeyCode[rc.Keymapping.R],
			Left:   keyNameToKeyCode[rc.Keymapping.Left],
			Right:  keyNameToKeyCode[rc.Keymapping.Right],
			Up:     keyNameToKeyCode[rc.Keymapping.Up],
			Down:   keyNameToKeyCode[rc.Keymapping.Down],
			Start:  keyNameToKeyCode[rc.Keymapping.Start],
			Select: keyNameToKeyCode[rc.Keymapping.Select],
		},
		WebRTC: rc.WebRTC,
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
