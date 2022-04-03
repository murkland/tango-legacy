package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/murkland/tango/config"
	"github.com/murkland/tango/mgba"
)

func ebitenToMgbaKeys(keymapping config.Keymapping, pressedKeys []ebiten.Key) mgba.Keys {
	keys := mgba.Keys(0)
	for _, key := range pressedKeys {
		if key == ebiten.Key(keymapping.A) {
			keys |= mgba.KeysA
		}
		if key == ebiten.Key(keymapping.B) {
			keys |= mgba.KeysB
		}
		if key == ebiten.Key(keymapping.L) {
			keys |= mgba.KeysL
		}
		if key == ebiten.Key(keymapping.R) {
			keys |= mgba.KeysR
		}
		if key == ebiten.Key(keymapping.Left) {
			keys |= mgba.KeysLeft
		}
		if key == ebiten.Key(keymapping.Right) {
			keys |= mgba.KeysRight
		}
		if key == ebiten.Key(keymapping.Up) {
			keys |= mgba.KeysUp
		}
		if key == ebiten.Key(keymapping.Down) {
			keys |= mgba.KeysDown
		}
		if key == ebiten.Key(keymapping.Start) {
			keys |= mgba.KeysStart
		}
		if key == ebiten.Key(keymapping.Select) {
			keys |= mgba.KeysSelect
		}
	}
	return keys
}
