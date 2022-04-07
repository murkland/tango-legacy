package config

import (
	"fmt"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pion/webrtc/v3"
)

type Keymapping struct {
	A         Key
	B         Key
	L         Key
	R         Key
	Left      Key
	Right     Key
	Up        Key
	Down      Key
	Start     Key
	Select    Key
	DebugSpew Key
}

type Netplay struct {
	InputDelay int
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

func (ait *AudioInterpolationType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "rubbery":
		*ait = AudioInterpolationTypeRubbery
	case "clippy":
		*ait = AudioInterpolationTypeClippy
	default:
		return fmt.Errorf("unknown audio interpolation type: %s", string(text))
	}
	return nil
}

func (ait AudioInterpolationType) MarshalText() ([]byte, error) {
	switch ait {
	case AudioInterpolationTypeRubbery:
		return []byte("rubbery"), nil
	case AudioInterpolationTypeClippy:
		return []byte("clippy"), nil
	default:
		return nil, fmt.Errorf("unknown audio interpolation type: %v", ait)
	}
}

type Config struct {
	Keymapping  Keymapping
	Audio       Audio
	Netplay     Netplay
	Matchmaking Matchmaking
	WebRTC      webrtc.Configuration
}

func Default() Config {
	return Config{
		Keymapping: Keymapping{
			A:         Key(ebiten.KeyZ),
			B:         Key(ebiten.KeyX),
			L:         Key(ebiten.KeyA),
			R:         Key(ebiten.KeyS),
			Left:      Key(ebiten.KeyArrowLeft),
			Right:     Key(ebiten.KeyArrowRight),
			Up:        Key(ebiten.KeyArrowUp),
			Down:      Key(ebiten.KeyArrowDown),
			Start:     Key(ebiten.KeyEnter),
			Select:    Key(ebiten.KeyBackspace),
			DebugSpew: Key(ebiten.KeyBackquote),
		},
		Audio: Audio{
			Interpolation: AudioInterpolationTypeClippy,
		},
		Netplay: Netplay{
			InputDelay: 3,
		},
		Matchmaking: Matchmaking{
			ConnectAddr: "mm.tango.murk.land:80",
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
}

func Save(config Config, w io.Writer) error {
	return toml.NewEncoder(w).Encode(config)
}

func Load(r io.Reader) (Config, error) {
	c := Default()

	if _, err := toml.NewDecoder(r).Decode(&c); err != nil {
		return c, err
	}

	return c, nil
}
