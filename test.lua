local FUN_battle_handleInput = 0x0801feee

local G_eLocalInput = 0x020399f2
local G_eRemoteInput = 0x02039a02

readu8 = memory.readbyteunsigned
reads8 = memory.readbyte
readu16 = memory.readshortunsigned
reads16 = memory.readshort
readu32 = memory.readlongunsigned
reads32 = memory.readlong
writeu8 = memory.writebyte
writes8 = memory.writebyte
writeu16 = memory.writeshort
writes16 = memory.writeshort
writeu32 = memory.writelong
writes32 = memory.writelong

local JOYPAD = {
    DEFAULT = 0xFC00,

    A = 0x0001,
    B = 0x0002,
    SELECT = 0x0004,
    START = 0x0008,
    RIGHT = 0x0010,
    LEFT = 0x0020,
    UP = 0x0040,
    DOWN = 0x0080,
    R = 0x0100,
    L = 0x0200,
}

memory.registerexec(
    FUN_battle_handleInput,
    function ()
        writeu16(G_eLocalInput, bit.bor(JOYPAD.DEFAULT, JOYPAD.RIGHT))
    end
)

function setBattleRNG(val)
    writeu32(0x020013f0, val)
end
