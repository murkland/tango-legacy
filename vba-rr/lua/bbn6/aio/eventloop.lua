local socket = require("socket")

local EventLoop = {}
EventLoop.__index = EventLoop

function EventLoop.new()
    local self = {
        running = false,
        read_callbacks = {},
        write_callbacks = {},
        callbacks = {},
    }

    setmetatable(self, EventLoop)
    return self
end

function EventLoop:add_read_callback(sock, cb)
    if self.read_callbacks[sock] == nil then
        self.read_callbacks[sock] = {}
    end
    self.read_callbacks[sock][#self.read_callbacks[sock]+1] = cb
end

function EventLoop:remove_read_socket(sock)
    self.read_callbacks[sock] = nil
end

function EventLoop:add_write_callback(sock, cb)
    if self.write_callbacks[sock] == nil then
        self.write_callbacks[sock] = {}
    end
    self.write_callbacks[sock][#self.write_callbacks[sock]+1] = cb
end

function EventLoop:remove_write_socket(sock)
    self.write_callbacks[sock] = nil
end

function EventLoop:_get_read_sockets()
    local sockets = {}
    for sock, _ in pairs(self.read_callbacks) do
        sockets[#sockets+1] = sock
    end
    return sockets
end

function EventLoop:_get_write_sockets()
    local sockets = {}
    for sock, _ in pairs(self.write_callbacks) do
        sockets[#sockets+1] = sock
    end
    return sockets
end

function EventLoop:_step()
    local readable, writable, err = socket.select(self:_get_read_sockets(), self:_get_write_sockets(), 0)

    for _, sock in ipairs(readable) do
        local cbs = self.read_callbacks[sock]
        self.read_callbacks[sock] = nil
        if cbs ~= nil then
            for _, cb in ipairs(cbs) do
                cb()
            end
        end
    end

    for _, sock in ipairs(writable) do
        local cbs = self.write_callbacks[sock]
        self.write_callbacks[sock] = nil
        if cbs ~= nil then
            for _, cb in ipairs(cbs) do
                cb()
            end
        end
    end

    local callbacks = self.callbacks
    self.callbacks = {}
    for _, cb in ipairs(callbacks) do
        cb()
    end
end

function EventLoop:add_callback(cb)
    self.callbacks[#self.callbacks+1] = cb
end

function EventLoop:run()
    self.running = true
    while self.running do
        self:_step()
    end
end

return EventLoop
