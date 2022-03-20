local socket = require("socket")
local struct = require("struct")

local PACKET_TYPE_INPUT = 1

local Client = {
    send_input = function (self, joyflags)
    end,

    recv = function(self)
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

    recv_input = function (self)
        local packet = self:recv()
        assert(packet.type == PACKET_TYPE_INPUT)
        return packet.joyflags
    end,

    send_marshaled_state = function (self, marshaled_state)
        print("would send marshaled state.")
    end
}

Client.__index = Client

function new_client(addr, port)
    local sock = assert(socket.connect(addr, port))
    local client = {sock = sock}
    setmetatable(client, Client)
    return client
end

return {
    new_client = new_client
}
