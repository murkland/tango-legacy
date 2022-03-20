local log = require("./log")
log.info("welcome to bingus battle network 6.")

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
    romoffsets.commMenu_handleLinkCableInput__entry,
    function ()
        log.error("unhandled call to SIO at 0x%08x: uh oh!", memory.read_reg("r14") - 1)
    end
)

memory.on_exec(
    romoffsets.battle_start__ret,
    function ()
        log.debug("battle started")
    end
)

memory.on_exec(
    romoffsets.battle_init__call__battle_copyInputData,
    function ()
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)
        -- This is the value at 0x0203F7D8 read earlier, set it to 0.
        memory.write_reg("r4", 0)
    end
)

memory.on_exec(
    romoffsets.battle_update__call__battle_copyInputData,
    function ()
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)
        -- This is the value at 0x0203F7D8 read earlier, set it to 0.
        memory.write_reg("r4", 0)

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
        log.debug("turn ended")
    end
)

memory.on_exec(
    romoffsets.battle_init_marshal__ret,
    function ()
        -- Inject code at the end of battle_custom_complete.
        log.debug("init ending")

        local state = battle.get_local_marshaled_state()
        client:send_marshaled_state(state)
        battle.set_player_marshaled_state(local_index, state)
        -- TODO: This has to be asynchronous.
        battle.set_player_marshaled_state(remote_index, client:recv_marshaled_state())
    end
)

memory.on_exec(
    romoffsets.battle_turn_marshal__ret,
    function ()
        -- Inject code at the end of battle_custom_complete.
        log.debug("turn resuming")

        local state = battle.get_local_marshaled_state()
        client:send_marshaled_state(state)
        battle.set_player_marshaled_state(local_index, state)
        -- TODO: This has to be asynchronous.
        battle.set_player_marshaled_state(remote_index, client:recv_marshaled_state())
    end
)

memory.on_exec(
    romoffsets.commMenu_inBattle__call__commMenu_handleLinkCableInput,
    function ()
        -- Skip the SIO call.
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)
    end
)

memory.on_exec(
    romoffsets.commMenu_waitForFriend__call__commMenu_handleLinkCableInput,
    function ()
        -- Skip the SIO call.
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)

        memory.write_reg("r0", 2)
    end
)

memory.on_exec(
    romoffsets.commMenu_connecting__call__commMenu_handleLinkCableInput,
    function ()
        -- TODO: Do we need to sync RNGs here?

        -- Skip the SIO call.
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)

        memory.write_reg("r0", 4)
    end
)

memory.on_exec(
    romoffsets.battle_isRemote__entry,
    function()
        memory.write_reg("r0", local_index)
        memory.write_reg("r15", memory.read_reg("r14")) -- mov lr, pc
    end
)

memory.on_exec(
    romoffsets.link_isRemote__entry,
    function()
        memory.write_reg("r0", local_index)
        memory.write_reg("r15", memory.read_reg("r14")) -- mov lr, pc
    end
)
