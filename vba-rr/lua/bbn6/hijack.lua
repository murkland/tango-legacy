local log = require("bbn6.log")
log.info("welcome to bingus battle network 6.")

local memory = require("bbn6.platform.require")("memory")
local emulator = require("bbn6.platform.require")("emulator")
local Client = require("bbn6.netplay")

local EventLoop = require("bbn6.aio.eventloop")
local romoffsets = require("bbn6.romoffsets")
local battle = require("bbn6.battle")

function hijack(sock, local_index)
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
            client:queue_turn(battle.get_active_in_battle_time(), local_turn)
        end
    )

    memory.on_exec(
        romoffsets.battle_getSettingsForLink__ret,
        function ()
            log.warn("UNIMPLEMENTED: a random battle settings pointer should be returned here")
        end
    )

    memory.on_exec(
        romoffsets.battle_start__ret,
        function ()
            log.debug("battle started")
        end
    )

    local last_tick = -1

    memory.on_exec(
        romoffsets.battle_end__entry,
        function ()
            last_tick = -1
            log.debug("battle ended")
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
            memory.write_reg("r0", 0xff)

            local local_tick = battle.get_active_in_battle_time()
            local local_joyflags = battle.get_local_joyflags()
            local local_custom_state = battle.get_local_custom_state()

            if last_tick < local_tick then
                if not client:queue_local_input(local_tick, local_joyflags, local_custom_state) then
                    -- log.warn("local input queue full, input processing will be stalled!")
                    return
                end
                last_tick = local_tick
            end

            local inputs = client:dequeue_inputs()

            if inputs == nil then
                -- log.warn("remote input is not available, input processing will be stalled!")
                return
            end
            -- Writing 0x0 to r0 indicates data is available.
            memory.write_reg("r0", 0x0)

            assert(inputs.tick + client.min_delay == local_tick, string.format("received tick != expected tick: %d != %d", inputs.tick + client.min_delay, local_tick))

            battle.set_rx_input_state(local_index, inputs.local_.joyflags, inputs.local_.custom_state)
            battle.set_rx_input_state(remote_index, inputs.remote.joyflags, inputs.remote.custom_state)

            if inputs.local_turn ~= nil then
                battle.set_rx_marshaled_state(local_index, inputs.local_turn)
                log.info("local turn committed on %df", local_tick)
            end

            if inputs.remote_turn ~= nil then
                battle.set_rx_marshaled_state(remote_index, inputs.remote_turn)
                log.info("remote turn committed on %df", local_tick)
            end
        end
    )

    memory.on_exec(
        romoffsets.battle_updating__ret__go_to_custom_screen,
        function ()
            log.debug("turn ended on %df", battle.get_active_in_battle_time())
        end
    )

    memory.on_exec(
        romoffsets.commMenu_waitForFriend__call__commMenu_handleLinkCableInput,
        function ()
            memory.write_reg("r15", memory.read_reg("r15") + 0x4)

            -- Just start the battle!
            memory.write_u8(0x02009a30 + 0x0, 0x18)
            memory.write_u8(0x02009a30 + 0x1, 0x18)
            memory.write_u8(0x02009a30 + 0x2, 0x00)
            memory.write_u8(0x02009a30 + 0x3, 0x00)
        end
    )

    memory.on_exec(
        romoffsets.commMenu_inBattle__call__commMenu_handleLinkCableInput,
        function ()
            memory.write_reg("r15", memory.read_reg("r15") + 0x4)
        end
    )

    log.info("hijack complete, starting event loop.")

    client:start(loop)
    loop:run()
end

return hijack
