local function table_copy(t)
    local u = { }
    for k, v in pairs(t) do u[k] = v end
    return setmetatable(u, getmetatable(t))
  end


local Client = {
    send_input = function (self, joyflags)
        self.joyflags = joyflags
    end,

    recv_input = function (self)
        return self.joyflags
    end,

    send_turn_commit = function (self, turn_commit)
        -- not sure what this business is, but it works
        self.turn_commit = table_copy(turn_commit)
        if self.player_index == 0 then
            self.turn_commit[0xb8 + 0x08 + 1] = 0xb0
            self.turn_commit[0xb8 + 0x09 + 1] = 0xa9
        else
            self.turn_commit[0xb8 + 0x08 + 1] = 0x88
            self.turn_commit[0xb8 + 0x09 + 1] = 0xaa
        end
    end,

    recv_turn_commit = function (self)
        return self.turn_commit
    end
}

Client.__index = Client

function new_client(player_index)
    local client = {
        player_index = player_index,
        joyflags = 0xfc00,
    }
    setmetatable(client, Client)
    return client
end

return {
    new_client = new_client
}
