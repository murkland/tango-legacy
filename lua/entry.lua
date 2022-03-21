local log = require("./log")
log.info("welcome to bingus battle network 6.")

local memory = require("./platform/require")("memory")
local emulator = require("./platform/require")("emulator")

local EventLoop = require("./aio/eventloop")
local romoffsets = require("./romoffsets")
local input = require("./input")
local battle = require("./battle")

function entry(Client, sock, local_index)
    local loop = EventLoop.new()

    local client = Client.new(sock)
    local remote_index = 1 - local_index

    memory.on_exec(
        romoffsets.commMenu_handleLinkCableInput__entry,
        function ()
            log.error("unhandled call to SIO at 0x%08x: uh oh!", memory.read_reg("r14") - 1)
        end
    )

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
        romoffsets.battle_init_marshal__ret,
        function ()
            local local_init = battle.get_tx_marshaled_state()
            client:give_init(local_init)
            battle.set_rx_marshaled_state(local_index, local_init)
            log.debug("init ending")
        end
    )

    memory.on_exec(
        romoffsets.battle_turn_marshal__ret,
        function ()
            local local_turn = battle.get_tx_marshaled_state()
            client:give_turn(local_turn)
            battle.set_rx_marshaled_state(local_index, local_turn)
            log.debug("turn resuming")
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
            local remote_init = client:take_init()
            if remote_init ~= nil then
                battle.set_rx_marshaled_state(remote_index, remote_init)
            end
        end
    )

    memory.on_exec(
        romoffsets.battle_update__call__battle_copyInputData,
        function ()
            memory.write_reg("r15", memory.read_reg("r15") + 0x4)

            if battle.get_state() == battle.State.CUSTOM_SCREEN then
                local remote_turn = client:take_turn()
                if remote_turn ~= nil then
                    battle.set_rx_marshaled_state(remote_index, remote_turn)
                    memory.write_reg("r0", 0x0)
                    return
                end
            end

            local local_tick = battle.get_active_in_battle_time()
            local local_joyflags = input.get_flags(0)
            battle.set_rx_joyflags(local_index, local_joyflags)

            if battle.get_state() == battle.State.IN_TURN then
                client:give_input(local_tick, local_joyflags)
            end

            local remote_input = client:take_input()

            if remote_input == nil and battle.get_state() == battle.State.IN_TURN then
                memory.write_reg("r0", 0xff)
                return
            end
            memory.write_reg("r0", 0x0)

            if remote_input == nil then
                return
            end

            if local_tick ~= remote_input.tick then
                log.fatal("local tick != remote tick: %d != %d", local_tick, remote_input.tick)
            end

            battle.set_rx_joyflags(remote_index, remote_input.joyflags)
        end
    )

    memory.on_exec(
        romoffsets.battle_updating__ret__go_to_custom_screen,
        function ()
            log.debug("turn ended")
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

    log.info("execution hijack complete, starting event loop.")

    client:start(loop)
    local function cb()
        emulator.advance_frame()
        loop:add_callback(cb)
    end
    cb()
    loop:run()
end

return entry
