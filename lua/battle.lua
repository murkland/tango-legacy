local memory = require("./platform/require")("memory")

local function set_battle_rng(val)
    memory.write_u32(0x020013f0, val)
end

local function set_p1_input(joyflags)
    memory.write_u16(0x02009488, joyflags)
end

local function set_p2_input(joyflags)
    memory.write_u16(0x0200948a, joyflags)
end

local function get_p1_hand(hand)
    return memory.read_range(0x020349c0, 0x50)
end

local function set_p1_hand(hand)
    assert(#hand == 0x50)
    memory.write_range(0x020349c0, hand)
end

local function get_p2_hand(hand)
    return memory.read_range(0x02034a10, 0x50)
end

local function set_p2_hand(hand)
    assert(#hand == 0x50)
    memory.write_range(0x02034a10, hand)
end

local function get_p1_navistats(navistats)
    return memory.read_range(0x0203ce00, 0x64)
end

local function set_p1_navistats(navistats)
    assert(#navistats == 0x64)
    memory.write_range(0x0203ce00, navistats)
end

local function get_p2_navistats(navistats)
    return memory.read_range(0x0203ce64, 0x64)
end

local function set_p2_navistats(navistats)
    assert(#navistats == 0x64)
    memory.write_range(0x0203ce64, navistats)
end

return {
    set_battle_rng = set_battle_rng,
    set_p1_input = set_p1_input,
    set_p2_input = set_p2_input,
    get_p1_hand = get_p1_hand,
    set_p1_hand = set_p1_hand,
    get_p2_hand = get_p2_hand,
    set_p2_hand = set_p2_hand,
    get_p1_navistats = get_p1_navistats,
    set_p1_navistats = set_p1_navistats,
    get_p2_navistats = get_p2_navistats,
    set_p2_navistats = set_p2_navistats,
}
