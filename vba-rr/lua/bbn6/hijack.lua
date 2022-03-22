local log = require("bbn6.log")
log.info("welcome to bingus battle network 6.")

local memory = require("bbn6.platform.require")("memory")
local emulator = require("bbn6.platform.require")("emulator")
local Client = require("bbn6.netplay")

local romoffsets = require("bbn6.romoffsets")
local battle = require("bbn6.battle")
local InputLog = require("bbn6.inputlog")

function hijack(sock, local_index)
    local client = Client.new(sock)
    local remote_index = 1 - local_index

    local battle_state = nil

    memory.on_exec(
        romoffsets.commMenu_handleLinkCableInput__entry,
        function ()
            log.error("unhandled call to SIO at 0x%08x: uh oh!", memory.read_reg("r14") - 1)
        end
    )

    memory.on_exec(
        romoffsets.battle_isRemote__tst,
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
        romoffsets.battle_start__ret,
        function ()
            log.debug("battle started: effects = %08x", battle.get_effects())
            battle_state = {
                input_log = InputLog.new(local_index),
                start_frame_number = nil,
                last_tick = -1,
            }
        end
    )

    memory.on_exec(
        romoffsets.battle_end__entry,
        function ()
            log.debug("battle ended")
            battle_state.input_log:close()
            battle_state = nil
        end
    )

    memory.on_exec(
        romoffsets.battle_init_marshal__ret,
        function ()
            local local_init = battle.get_local_marshaled_state()
            client:give_init(local_init)
            battle.set_player_marshaled_state(local_index, local_init)
            battle_state.input_log:write_init(false, local_init)
            log.debug("init ending")
        end
    )

    memory.on_exec(
        romoffsets.battle_turn_marshal__ret,
        function ()
            local local_turn = battle.get_local_marshaled_state()
            client:queue_local_turn(emulator.current_frame_number() - battle_state.start_frame_number, local_turn)
        end
    )

    memory.on_exec(
        romoffsets.battle_getSettingsForLink__ret,
        function ()
            log.warn("UNIMPLEMENTED: a random battle settings pointer should be returned here")
        end
    )

    memory.on_exec(
        romoffsets.battle_init__call__battle_copyInputData,
        function ()
            memory.write_reg("r15", memory.read_reg("r15") + 0x4)

            memory.write_reg("r0", 0x0)
            local remote_init = client:take_init()
            if remote_init ~= nil then
                battle_state.input_log:write_init(true, remote_init)
                battle.set_player_marshaled_state(remote_index, remote_init)
            end
        end
    )

    memory.on_exec(
        romoffsets.battle_update__call__battle_copyInputData,
        function ()
            if battle_state.start_frame_number == nil then
                battle_state.start_frame_number = emulator.current_frame_number()
            end

            memory.write_reg("r15", memory.read_reg("r15") + 0x4)
            memory.write_reg("r0", 0x00)

            local local_tick = emulator.current_frame_number() - battle_state.start_frame_number
            local local_joyflags = battle.get_local_joyflags()
            local local_custom_state = battle.get_local_custom_state()

            if battle_state.last_tick < local_tick then
                client:queue_local_input(local_tick, local_joyflags, local_custom_state)
                battle_state.last_tick = local_tick
            end

            local inputs = client:dequeue_inputs()

            assert(inputs.tick + client.delay == local_tick, string.format("received tick != expected tick: %d != %d", inputs.tick + client.delay, local_tick))

            battle_state.input_log:write_input(battle.get_rng2_state(), inputs)

            battle.set_player_input_state(local_index, inputs.local_.joyflags, inputs.local_.custom_state)
            battle.set_player_input_state(remote_index, inputs.remote.joyflags, inputs.remote.custom_state)

            if inputs.local_turn ~= nil then
                battle.set_player_marshaled_state(local_index, inputs.local_turn)
                log.info("local turn committed on %df", local_tick)
            end

            if inputs.remote_turn ~= nil then
                battle.set_player_marshaled_state(remote_index, inputs.remote_turn)
                log.info("remote turn committed on %df", local_tick)
            end
        end
    )

    memory.on_exec(
        romoffsets.battle_updating__ret__go_to_custom_screen,
        function ()
            log.debug("turn ended on %df, rng state = %08x", emulator.current_frame_number() - battle_state.start_frame_number, battle.get_rng2_state())
        end
    )

    memory.on_exec(
        romoffsets.commMenu_waitForFriend__call__commMenu_handleLinkCableInput,
        function ()
            memory.write_reg("r15", memory.read_reg("r15") + 0x4)
            battle.start_from_comm_menu()
        end
    )

    memory.on_exec(
        romoffsets.commMenu_inBattle__call__commMenu_handleLinkCableInput,
        function ()
            memory.write_reg("r15", memory.read_reg("r15") + 0x4)
        end
    )

    log.info("hijack complete >:)")
end

return hijack
