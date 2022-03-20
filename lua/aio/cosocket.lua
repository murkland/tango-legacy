local coroutine = require("coroutine")

local Cosocket = {}
Cosocket.__index = Cosocket

function Cosocket.new(sock, loop)
    local self = {sock = sock, loop = loop}
    self.sock:settimeout(0, "b")
    setmetatable(self, Cosocket)
    return self
end

function Cosocket:send(data, i, j)
    local co = coroutine.running()
    self.loop.add_write_callback(self.sock, function ()
        coroutine.resume(co, self.sock:send(data, i, j))
    end)
    return coroutine.yield()
end

function Cosocket:receive(pattern)
    local co = coroutine.running()
    self.loop.add_read_callback(self.sock, function ()
        coroutine.resume(co, self.sock:receive(pattern))
    end)
    return coroutine.yield()
end

return Cosocket
