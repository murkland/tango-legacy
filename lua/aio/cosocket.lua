local coroutine = require("coroutine")
local socket = require("socket")

local Cosocket = {}
Cosocket.__index = Cosocket

function Cosocket.new(sock)
    local self = {sock = sock}
    self.sock:settimeout(0, "b")
    setmetatable(self, Cosocket)
    return self
end

function Cosocket:send(loop, data, i, j)
    local co = coroutine.running()
    loop:add_write_callback(self.sock, function ()
        assert(coroutine.resume(co, self.sock:send(data, i, j)))
    end)
    return coroutine.yield()
end

function Cosocket:receive(loop, pattern)
    local co = coroutine.running()
    loop:add_read_callback(self.sock, function ()
        local r = self.sock:receive(pattern)
        if r == nil then
            -- No input received, reschedule for read.
            self:receive(loop, pattern)
            return
        end
        assert(coroutine.resume(co, r))
    end)
    return coroutine.yield()
end

function Cosocket:readable()
    local readable, _, err = socket.select({self.sock}, {}, 0)
    if err ~= nil then
        return false, err
    end
    return #readable == 1
end

function Cosocket:writable()
    local _, writable, err = socket.select({}, {self.sock}, 0)
    if err ~= nil then
        return false, err
    end
    return #writable == 1
end

return Cosocket
