local memory = require("./platform/require")("memory")

local romoffsets = require("./romoffsets")
local input = require("./input")
local battle = require("./battle")

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
        battle.set_player_input(0, inpflags)
        battle.set_player_input(1, inpflags)
    end
)
