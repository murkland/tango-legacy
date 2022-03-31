package bn6

import (
	"github.com/murkland/bbn6/mgba"
)

type BN6 struct {
	Offsets Offsets
}

func Load(romTitle string) *BN6 {
	offsets, ok := offsetsMap[romTitle]
	if !ok {
		return nil
	}
	return &BN6{offsets}
}

func (b *BN6) StartBattleFromCommMenu(core *mgba.Core) {
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x0, -1, 0x18)
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x1, -1, 0x18)
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x2, -1, 0x00)
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x3, -1, 0x00)
}

func (b *BN6) DropMatchmakingFromCommMenu(core *mgba.Core) {
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x0, -1, 0x18)
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x1, -1, 0x3c)
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x2, -1, 0x04)
	core.RawWrite8(b.Offsets.EWRAM.A_MenuControl+0x3, -1, 0x04)
}

func (b *BN6) LocalJoyflags(core *mgba.Core) uint16 {
	return core.RawRead16(b.Offsets.EWRAM.A_Joypad+0x00, -1)
}

func (b *BN6) LocalCustomScreenState(core *mgba.Core) uint8 {
	return core.RawRead8(b.Offsets.EWRAM.A_BattleState+0x11, -1)
}

func (b *BN6) LocalMarshaledBattleState(core *mgba.Core) []byte {
	var buf [0x100]byte
	core.RawReadRange(b.Offsets.EWRAM.A_LocalMarshaledBattleState, -1, buf[:])
	return buf[:]
}

func (b *BN6) SetPlayerInputState(core *mgba.Core, index int, keysPressed uint16, customScreenState uint8) {
	ePlayerInput := b.Offsets.EWRAM.A_PlayerInputDataArr + uint32(index)*0x08
	keysHeld := core.RawRead16(ePlayerInput+0x02, -1)
	core.RawWrite16(ePlayerInput+0x02, -1, keysPressed)
	core.RawWrite16(ePlayerInput+0x04, -1, ^keysHeld&keysHeld)
	core.RawWrite16(ePlayerInput+0x06, -1, keysHeld&^keysPressed)
	core.RawWrite8(b.Offsets.EWRAM.A_BattleState+0x14+uint32(index), -1, customScreenState)
}

func (b *BN6) SetPlayerMarshaledBattleState(core *mgba.Core, index int, marshaledState []byte) {
	core.RawWriteRange(b.Offsets.EWRAM.A_PlayerMarshaledStateArr+uint32(index)*0x100, -1, marshaledState)
}

func (b *BN6) LocalWins(core *mgba.Core) uint8 {
	return core.RawRead8(b.Offsets.EWRAM.A_BattleState+0x18, -1)
}

func (b *BN6) RemoteWins(core *mgba.Core) uint8 {
	return core.RawRead8(b.Offsets.EWRAM.A_BattleState+0x19, -1)
}

func (b *BN6) RNG2State(core *mgba.Core) uint32 {
	return core.RawRead32(b.Offsets.EWRAM.A_Rng2, -1)
}

func (b *BN6) MenuControlState(core *mgba.Core, offset uint32) uint8 {
	return core.RawRead8(b.Offsets.EWRAM.A_MenuControl+offset, -1)
}

func (b *BN6) SetLinkBattleSettingsAndBackground(core *mgba.Core, linkBattleSettingsAndBackground uint16) {
	core.RawWrite16(b.Offsets.EWRAM.A_MenuControl+0x2a, -1, linkBattleSettingsAndBackground)
}

func (b *BN6) BattleType(core *mgba.Core) uint8 {
	return core.RawRead8(b.Offsets.EWRAM.A_MenuControl+0x12, -1)
}
