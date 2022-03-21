local input = {}

local joypad = require("bbn6.platform.require")("joypad")

input.Joyflag = {
    DEFAULT = 0xFC00,

    A       = 0x0001,
    B       = 0x0002,
    select  = 0x0004,
    start   = 0x0008,
    right   = 0x0010,
    left    = 0x0020,
    up      = 0x0040,
    down    = 0x0080,
    R       = 0x0100,
    L       = 0x0200,
}

function input.get_flags(i)
    local flags = input.Joyflag.DEFAULT
    for k, v in pairs(joypad.get(i)) do
        if v then
            flags = bit.bor(flags, input.Joyflag[k])
        end
    end
    return flags
end

return input
