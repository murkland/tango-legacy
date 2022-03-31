package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/bbn6/config"
	"github.com/murkland/bbn6/mgba"
)

func ebitenToMgbaKeys(keymapping config.Keymapping, pressedKeys []ebiten.Key) mgba.Keys {
	keys := mgba.Keys(0xfc00)
	for _, key := range pressedKeys {
		if key == keymapping.A {
			keys |= mgba.KeysA
		}
		if key == keymapping.B {
			keys |= mgba.KeysB
		}
		if key == keymapping.L {
			keys |= mgba.KeysL
		}
		if key == keymapping.R {
			keys |= mgba.KeysR
		}
		if key == keymapping.Left {
			keys |= mgba.KeysLeft
		}
		if key == keymapping.Right {
			keys |= mgba.KeysRight
		}
		if key == keymapping.Up {
			keys |= mgba.KeysUp
		}
		if key == keymapping.Down {
			keys |= mgba.KeysDown
		}
		if key == keymapping.Start {
			keys |= mgba.KeysStart
		}
		if key == keymapping.Select {
			keys |= mgba.KeysSelect
		}
	}
	return keys
}
