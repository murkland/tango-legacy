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

-- TODO: Find where custom screen init is committed.

local function get_p1_custominit(custominit)
    return memory.read_range(0x0203f4a0, 0x100)
end

local function set_p1_custominit(custominit)
    assert(#custominit == 0x100)
    memory.write_range(0x0203f4a0, custominit)
end

local function get_p2_custominit(custominit)
    return memory.read_range(0x0203f5a0, 0x100)
end

local function set_p2_custominit(custominit)
    assert(#custominit == 0x100)
    memory.write_range(0x0203f5a0, custominit)
end

return {
    set_battle_rng = set_battle_rng,
    set_p1_input = set_p1_input,
    set_p2_input = set_p2_input,
    get_p1_custominit = get_p1_custominit,
    set_p1_custominit = set_p1_custominit,
    get_p2_custominit = get_p2_custominit,
    set_p2_custominit = set_p2_custominit,
}
