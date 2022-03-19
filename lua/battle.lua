local memory = require("./platform/require")("memory")

local function set_battle_rng(val)
    memory.write_u32(0x020013f0, val)
end

local gPlayerInputAddr = 0x02036820

local function set_player_input(index, keys_pressed) -- input_setPlayerInputKeys @ 0x0800a0d6
    local keys_held = memory.read_u16(gPlayerInputAddr + index * 8 + 2)
    memory.write_u16(gPlayerInputAddr + index * 8 + 2 --[[ keys_held ]], keys_pressed)
    memory.write_u8(gPlayerInputAddr + index * 8 + 4 --[[ keys_pressed ]], bit.band(keys_pressed, bit.bnot(keys_held)))
    memory.write_u8(gPlayerInputAddr + index * 8 + 6 --[[ keys_up ]], bit.band(keys_held, bit.bnot(keys_pressed)))
end

-- TODO: Find where custom screen init is committed.

local function get_player_custominit(index, custominit)
    return memory.read_range(0x0203f4a0, 0x100)
end

local function set_player_custominit(index, custominit)
    assert(#custominit == 0x100)
    memory.write_range(0x0203f4a0 + index * 0x100, custominit)
end

return {
    set_battle_rng = set_battle_rng,
    set_player_input = set_player_input,
    get_player_custominit = get_player_custominit,
    set_player_custominit = set_player_custominit,
}
