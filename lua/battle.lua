local memory = require("./platform/require")("memory")

local g_player_input_attr = 0x02036820

local function set_player_input(index, keys_pressed) -- input_setPlayerInputKeys @ 0x0800a0d6
    local keys_held = memory.read_u16(g_player_input_attr + index * 8 + 2)
    memory.write_u16(g_player_input_attr + index * 8 + 2 --[[ keys_held ]], keys_pressed)
    memory.write_u8(g_player_input_attr + index * 8 + 4 --[[ keys_pressed ]], bit.band(keys_pressed, bit.bnot(keys_held)))
    memory.write_u8(g_player_input_attr + index * 8 + 6 --[[ keys_up ]], bit.band(keys_held, bit.bnot(keys_pressed)))
end

local function set_rx_marshaled_state(index, marshaled_state)
    assert(#marshaled_state == 0x100)
    memory.write_range(0x0203f4a0 + index * 0x100, marshaled_state)
end

local function get_tx_marshaled_state(index, marshaled_state)
    return memory.read_range(0x0203cbe0, 0x100)
end

local function get_battle_state()
    return memory.read_u8(0x02034880 + 0x1)
end

local function is_in_custom_screen()
    return get_battle_state() == 8
end

local function is_in_turn()
    return get_battle_state() == 12
end

local function get_elapsed_active_time()
    return memory.read_u16(0x020348c0)
end

return {
    set_player_input = set_player_input,
    set_rx_marshaled_state = set_rx_marshaled_state,
    get_tx_marshaled_state = get_tx_marshaled_state,
    is_in_custom_screen = is_in_custom_screen,
    is_in_turn = is_in_turn,
    get_elapsed_active_time = get_elapsed_active_time,
}
