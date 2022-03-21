local battle = {}

local memory = require("bbn6.platform.require")("memory")

local e_player_input_data_arr = 0x02036820
local e_battle_state = 0x02034880
local e_joypad = 0x0200a270
local e_local_marshaled_state = 0x0203cbe0
local e_player_marshaled_state_arr = 0x0203f4a0
local e_menu_control = 0x02009a30

function battle.start_from_comm_menu()
    memory.write_u8(e_menu_control + 0x0, 0x18)
    memory.write_u8(e_menu_control + 0x1, 0x18)
    memory.write_u8(e_menu_control + 0x2, 0x00)
    memory.write_u8(e_menu_control + 0x3, 0x00)
end

function battle.get_local_joyflags()
    return memory.read_u16(e_joypad + 0x00)
end

function battle.get_local_custom_state()
    return memory.read_u8(e_battle_state + 0x11)
end

function battle.get_local_marshaled_state(index, marshaled_state)
    return memory.read_range(e_local_marshaled_state, 0x100)
end

function battle.set_player_input_state(index, keys_pressed, custom_state) -- joyflags_setPlayerInputKeys @ 0x0800a0d6
    local keys_held = memory.read_u16(e_player_input_data_arr + index * 8 + 0x02)
    memory.write_u16(e_player_input_data_arr + index * 8 + 0x02 --[[ keys_held ]], keys_pressed)
    memory.write_u8(e_player_input_data_arr + index * 8 + 0x04 --[[ keys_pressed ]], bit.band(keys_pressed, bit.bnot(keys_held)))
    memory.write_u8(e_player_input_data_arr + index * 8 + 0x06 --[[ keys_up ]], bit.band(keys_held, bit.bnot(keys_pressed)))

    -- Set player custom state.
    memory.write_u8(e_battle_state + 0x14 + index, custom_state)
end

function battle.set_player_marshaled_state(index, marshaled_state)
    assert(#marshaled_state == 0x100)
    memory.write_range(e_player_marshaled_state_arr + index * 0x100, marshaled_state)
end

function battle.get_active_in_battle_time()
    return memory.read_u32(e_battle_state + 0x64)
end

return battle
