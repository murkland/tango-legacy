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

local g_rx_marshaled_state_ptr = 0x0203f4a0

local function set_rx_marshaled_state(index, marshaled_state)
    assert(#marshaled_state == 0x100)
    memory.write_range(g_rx_marshaled_state_ptr + index * 0x100, marshaled_state)
end

local g_tx_marshaled_state_ptr = 0x0203cbe0

local function get_tx_marshaled_state(index, marshaled_state)
    return memory.read_range(g_tx_marshaled_state_ptr, 0x100)
end

local g_battle_state = 0x02034880

local function is_in_custom_screen()
    return memory.read_u8(g_battle_state + 0x1) == 8
end

return {
    set_battle_rng = set_battle_rng,
    set_player_input = set_player_input,
    set_rx_marshaled_state = set_rx_marshaled_state,
    get_tx_marshaled_state = get_tx_marshaled_state,
    is_in_custom_screen = is_in_custom_screen,
}
