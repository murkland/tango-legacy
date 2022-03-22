local io = require("io")
local struct = require("bbn6.struct")
local log = require("bbn6.log")

function u8table_to_string(t)
    local s = {}
    for i, c in ipairs(t) do
        s[i] = string.char(c)
    end
    return table.concat(s)
end

local InputLog = {}
InputLog.__index = InputLog

function InputLog.new(local_index)
    local fn = "input_p" .. (local_index + 1) .. "_" .. os.date("%Y%m%d%H%M%S") .. ".log"
    local f = io.open(fn, "w")
    log.info("starting input log to %s", fn)
    local self = {f = f, local_index = local_index}
    setmetatable(self, InputLog)
    return self
end

function InputLog:_write_line(line)
    self.f:write(line)
    self.f:write("\n")
    self.f:flush()
end

function InputLog:write_init(is_remote, init)
    local local_name = "p1"
    local remote_name = "p2"

    if self.local_index == 1 then
        local_name = "p2"
        remote_name = "p1"
    end

    local player = local_name
    if is_remote then
        player = remote_name
    end
    self:_write_line(string.format("INIT %s: %s", player, struct.hexdump(u8table_to_string(init))))
end

function InputLog:write_input(rng_state, inputs)
    local p1_state = inputs.local_
    local p2_state = inputs.remote
    local local_name = "p1"
    local remote_name = "p2"

    if self.local_index == 1 then
        p1_state = inputs.remote
        p2_state = inputs.local_
        local_name = "p2"
        remote_name = "p1"
    end

    self:_write_line(string.format(
        "%df: rng=%08x p1_joyflags=%04x p2_joyflags=%04x p1_custom_state=%d p2_custom_state=%d",
        inputs.tick,
        rng_state,
        p1_state.joyflags, p2_state.joyflags,
        p1_state.custom_state, p2_state.custom_state
    ))
    if inputs.local_turn ~= nil then
        self:_write_line(string.format("  + %s turn: %s", local_name, struct.hexdump(u8table_to_string(inputs.local_turn))))
    end
    if inputs.remote_turn ~= nil then
        self:_write_line(string.format("  + %s turn: %s", remote_name, struct.hexdump(u8table_to_string(inputs.remote_turn))))
    end
end

function InputLog:close()
    log.info("closing input log")
    self.f:close()
end

return InputLog
