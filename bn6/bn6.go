package bn6

import (
	"github.com/murkland/bbn6/mgba"
)

const (
	ePlayerInputDataArr        = 0x02036820
	eBattleState               = 0x02034880
	eJoypad                    = 0x0200a270
	eLocalMarshaledBattleState = 0x0203cbe0
	ePlayerMarshaledStateArr   = 0x0203f4a0
	eMenuControl               = 0x02009a30
	eRng2                      = 0x020013f0
)

func StartBattleFromCommMenu(core *mgba.Core) {
	core.RawWrite8(eMenuControl+0x0, -1, 0x18)
	core.RawWrite8(eMenuControl+0x1, -1, 0x18)
	core.RawWrite8(eMenuControl+0x2, -1, 0x00)
	core.RawWrite8(eMenuControl+0x3, -1, 0x00)
}

func LocalJoyflags(core *mgba.Core) uint16 {
	return core.RawRead16(eJoypad+0x00, -1)
}

func SetLocalJoyflags(core *mgba.Core, joyflags uint16) {
	core.RawWrite16(eJoypad+0x00, -1, joyflags)
}

func LocalCustomScreenState(core *mgba.Core) uint8 {
	return core.RawRead8(eBattleState+0x11, -1)
}

func LocalMarshaledBattleState(core *mgba.Core) []byte {
	var buf [0x100]byte
	core.RawReadRange(eLocalMarshaledBattleState, -1, buf[:])
	return buf[:]
}

func SetPlayerInputState(core *mgba.Core, index int, keysPressed uint16, customScreenState uint8) {
	ePlayerInput := uint32(ePlayerInputDataArr + index*0x08)
	keysHeld := core.RawRead16(ePlayerInput+0x02, -1)
	core.RawWrite16(ePlayerInput+0x02, -1, keysPressed)
	core.RawWrite16(ePlayerInput+0x04, -1, ^keysHeld&keysHeld)
	core.RawWrite16(ePlayerInput+0x06, -1, keysHeld&^keysPressed)
	core.RawWrite8(uint32(eBattleState+0x14+index), -1, customScreenState)
}

func SetPlayerMarshaledBattleState(core *mgba.Core, index int, marshaledState []byte) {
	core.RawWriteRange(uint32(ePlayerMarshaledStateArr+index*0x100), -1, marshaledState)
}

func RNG2State(core *mgba.Core) uint32 {
	return core.RawRead32(eRng2, -1)
}
