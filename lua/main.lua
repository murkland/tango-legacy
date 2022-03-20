local memory = require("./platform/require")("memory")

local romoffsets = require("./romoffsets")
local input = require("./input")
local battle = require("./battle")

local netplay_dummy = require("./netplay_dummy")

-- TODO: Dynamically initialize this.
local local_index = 1
local remote_index = 1 - local_index

local client = netplay_dummy.new_client(local_index)

memory.on_exec(
    romoffsets.battle_handleLinkCableInput__call__battle_handleLinkSIO,
    function ()
        -- Stub out the call to battle_handleLinkSIO: we will be handling IO on our own here without involving SIO.
        -- The next 4 bytes is a call to battle_handleLinkSIO, which expects r0 to be 0x2 if input is ready. Input is always ready, so we just skip the call and write the appropriate value.
        memory.write_reg("r0", 0x2)
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)
    end
)

memory.on_exec(
    romoffsets.battle_update__call__battle_copyInputData,
    function ()
        -- Stub out the call to battle_copyInputData: this handles setting the input and copying CustomInit data in 32-bit chunks.
        -- We're going to handle all of this ourselves, so no need to run this function.
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)

        local inpflags = input.get_flags(0)
        if not battle.is_in_custom_screen() then
            client:send_input(inpflags)
        end

        battle.set_player_input(local_index, inpflags)
        battle.set_player_input(remote_index, client:recv_input())
    end
)

memory.on_exec(
    romoffsets.battle_updating__ret__go_to_custom_screen,
    function ()
        print("DEBUG: turn ended")
    end
)

memory.on_exec(
    romoffsets.battle_custom_complete__ret,
    function ()
        -- Inject code at the end of battle_custom_complete.
        local tc = battle.get_local_turn_commit()
        client:send_turn_commit(tc)
        print("DEBUG: turn resuming")

        battle.set_player_turn_commit(local_index, tc)

        -- TODO: This has to be asynchronous.
        battle.set_player_turn_commit(remote_index, client:recv_turn_commit())
    end
)
