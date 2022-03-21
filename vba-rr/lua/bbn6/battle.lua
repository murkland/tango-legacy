local battle = {}

local memory = require("bbn6.platform.require")("memory")

local g_player_joyflags_attr = 0x02036820

function battle.set_rx_joyflags(index, keys_pressed) -- joyflags_setPlayerInputKeys @ 0x0800a0d6
    local keys_held = memory.read_u16(g_player_joyflags_attr + index * 8 + 2)
    memory.write_u16(g_player_joyflags_attr + index * 8 + 2 --[[ keys_held ]], keys_pressed)
    memory.write_u8(g_player_joyflags_attr + index * 8 + 4 --[[ keys_pressed ]], bit.band(keys_pressed, bit.bnot(keys_held)))
    memory.write_u8(g_player_joyflags_attr + index * 8 + 6 --[[ keys_up ]], bit.band(keys_held, bit.bnot(keys_pressed)))
end

function battle.set_rx_marshaled_state(index, marshaled_state)
    assert(#marshaled_state == 0x100)
    memory.write_range(0x0203f4a0 + index * 0x100, marshaled_state)
end

function battle.get_tx_marshaled_state(index, marshaled_state)
    return memory.read_range(0x0203cbe0, 0x100)
end

function battle.get_state()
    return memory.read_u8(0x02034880 + 0x1)
end

function battle.get_active_unpaused_time()
    return memory.read_u16(0x020348c0)
end

function battle.get_active_total_time()
    return memory.read_u32(0x020348e0)
end

function battle.get_active_in_battle_time()
    return memory.read_u32(0x020348e4)
end

battle.State = {
    INIT = 0,
    CUSTOM_SCREEN = 8,
    IN_TURN = 12,
}

return battle
