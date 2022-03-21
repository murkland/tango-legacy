local io = require("io")

local InputLog = {}
InputLog.__index = InputLog

local function dump(o)
    if type(o) == 'table' then
       local s = '{ '
       for k,v in pairs(o) do
          if type(k) ~= 'number' then k = '"'..k..'"' end
          s = s .. '['..k..'] = ' .. dump(v) .. ','
       end
       return s .. '} '
    else
       return tostring(o)
    end
 end

function InputLog.new()
    local self = {f = io.open("input.log", "w")}
    setmetatable(self, InputLog)
    return self
end

function InputLog:_write_line(line)
    self.f:write(line)
    self.f:write("\n")
    self.f:flush()
end

function InputLog:write_init(is_remote, init)
    local turn_type = "LOCAL"
    if is_remote then
        turn_type = "REMOTE"
    end
    self:_write_line(string.format("%s INIT: %s", turn_type, dump(init)))
end

function InputLog:write_input(tick, rng_state, inputs)
    self:_write_line(string.format("%df: rng=%08x %s", tick, rng_state, dump(inputs)))
end

function InputLog:close()
    self.f:close()
end

return InputLog
