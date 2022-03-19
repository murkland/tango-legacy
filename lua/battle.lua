local memory = require("./platform/require")("memory")

local function set_battle_rng(val)
    memory.write_u32(0x020013f0, val)
end

local g_player_input_attr = 0x02036820

local function set_player_input(index, keys_pressed) -- input_setPlayerInputKeys @ 0x0800a0d6
    local keys_held = memory.read_u16(g_player_input_attr + index * 8 + 2)
    memory.write_u16(g_player_input_attr + index * 8 + 2 --[[ keys_held ]], keys_pressed)
    memory.write_u8(g_player_input_attr + index * 8 + 4 --[[ keys_pressed ]], bit.band(keys_pressed, bit.bnot(keys_held)))
    memory.write_u8(g_player_input_attr + index * 8 + 6 --[[ keys_up ]], bit.band(keys_held, bit.bnot(keys_pressed)))
end

-- TODO: Find where custom screen init is committed.

local g_player_turn_commit_ptr = 0x0203f4a0

local function get_player_turn_commit(index, turn_commit)
    return memory.read_range(g_player_turn_commit_ptr, 0x100)
end

local function set_player_turn_commit(index, turn_commit)
    assert(#turn_commit == 0x100)
    memory.write_range(g_player_turn_commit_ptr + index * 0x100, turn_commit)
end

local g_local_turn_commit_ptr = 0x0203cbe0

local function get_local_turn_commit(index, turn_commit)
    return memory.read_range(g_local_turn_commit_ptr, 0x100)
end

return {
    set_battle_rng = set_battle_rng,
    set_player_input = set_player_input,
    get_player_turn_commit = get_player_turn_commit,
    set_player_turn_commit = set_player_turn_commit,
    get_local_turn_commit = get_local_turn_commit,
}
