local socket = require("socket")
local struct = require("struct")

local PACKET_TYPE_INPUT = 1

local Client = {}
Client.__index = Client

function Client.new(addr, port)
    local sock = assert(socket.connect(addr, port))
    local client = {sock = sock}
    setmetatable(client, Client)
    return client
end

function Client:send_input(joyflags)
end

function Client:recv()
    local type = assert(self.sock:receive(1)):byte(1)
    if type == PACKET_TYPE_INPUT then
        local body = struct.read(assert(self.sock:receive(2)), "w")
        return {
            type = type,
            joyflags = body[1],
        }
    end

    assert(false, "unknown packet type: " .. type)
end,

function Client:recv_input()
    local packet = self:recv()
    assert(packet.type == PACKET_TYPE_INPUT)
    return packet.joyflags
end,

function Client:send_marshaled_state(marshaled_state)
    print("would send marshaled state.")
end

return Client
