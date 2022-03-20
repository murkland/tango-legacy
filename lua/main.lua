local log = require("./log")
log.info("welcome to bingus battle network 6.")

local memory = require("./platform/require")("memory")
local emulator = require("./platform/require")("emulator")

local EventLoop = require("./eventloop")
local romoffsets = require("./romoffsets")
local input = require("./input")
local battle = require("./battle")

local Client = require("./netplay_dummy")

-- TODO: Dynamically initialize this.
local local_index = 0
local remote_index = 1 - local_index

local client = Client.new(local_index)

memory.on_exec(
    romoffsets.battle_isRemote__ret,
    function()
        memory.write_reg("r0", local_index)
    end
)

memory.on_exec(
    romoffsets.link_isRemote__ret,
    function()
        memory.write_reg("r0", local_index)
    end
)

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
        memory.write_reg("r0", 0x0)
    end
)

memory.on_exec(
    romoffsets.battle_init_marshal__ret,
    function ()
        -- Inject code at the end of battle_custom_complete.
        log.debug("init ending")

        local local_init = battle.get_tx_marshaled_state()
        client:send_init(local_init)
        battle.set_rx_marshaled_state(local_index, local_init)

        local remote_init = client:take_init()
        if remote_init == nil then
            return
        end
        battle.set_rx_marshaled_state(remote_index, remote_init)
    end
)

memory.on_exec(
    romoffsets.battle_update__call__battle_copyInputData,
    function ()
        local local_input = input.get_flags(0)
        battle.set_player_input(local_index, local_input)

        if battle.is_in_turn() then
            client:send_input(local_input)
        end

        local remote_input = client:take_input()

        memory.write_reg("r15", memory.read_reg("r15") + 0x4)
        if remote_input == nil and battle.is_in_turn() then
            memory.write_reg("r0", 0xff)
            return
        end
        memory.write_reg("r0", 0x0)

        if remote_input == nil then
            return
        end
        battle.set_player_input(remote_index, remote_input)
    end
)

memory.on_exec(
    romoffsets.battle_updating__ret__go_to_custom_screen,
    function ()
        log.debug("turn ended")
    end
)

memory.on_exec(
    romoffsets.battle_turn_marshal__ret,
    function ()
        -- Inject code at the end of battle_custom_complete.
        log.debug("turn resuming")

        local local_turn = battle.get_tx_marshaled_state()
        client:send_turn(local_turn)
        battle.set_rx_marshaled_state(local_index, local_turn)

        local remote_turn = client:take_turn()
        if remote_turn == nil then
            return
        end
        battle.set_rx_marshaled_state(remote_index, remote_turn)
    end
)

memory.on_exec(
    romoffsets.commMenu_waitForFriend__call__commMenu_handleLinkCableInput,
    function ()
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)

        -- Just start the battle!
        memory.write_u8(0x02009a31, 0x18)
        memory.write_u8(0x02009a32, 0x00)
        memory.write_u8(0x02009a33, 0x00)
    end
)

memory.on_exec(
    romoffsets.commMenu_inBattle__call__commMenu_handleLinkCableInput,
    function ()
        memory.write_reg("r15", memory.read_reg("r15") + 0x4)
    end
)

local loop = EventLoop.new()
function run_emu()
    emulator.advance_frame()
    loop:add_callback(run_emu)
end
run_emu()
loop:run()
