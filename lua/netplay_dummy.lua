local Client = {
    send_input = function (self, joyflags)
        self.joyflags = joyflags
    end,

    recv_input = function (self)
        return self.joyflags
    end,

    send_turn_commit = function (self, turn_commit)
        self.turn_commit = turn_commit
    end,

    recv_turn_commit = function (self, turn_commit)
        return self.turn_commit
    end
}

Client.__index = Client

function new_client(addr, port)
    local client = {
        joyflags = 0xfc00,
    }
    setmetatable(client, Client)
    return client
end

return {
    new_client = new_client
}
