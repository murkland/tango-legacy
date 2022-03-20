local function table_copy(t)
    local u = { }
    for k, v in pairs(t) do u[k] = v end
    return setmetatable(u, getmetatable(t))
  end

local Client = {}
Client.__index = Client

function Client.new(player_index)
    local client = {
        player_index = player_index,
        joyflags = 0xfc00,
    }
    setmetatable(client, Client)
    return client
end

function Client:send_input(joyflags)
    self.joyflags = joyflags
end


function Client:recv_input()
    return self.joyflags
end

function Client:send_marshaled_state(marshaled_state)
    -- not sure what this business is, but it works
    self.marshaled_state = table_copy(marshaled_state)
    if self.player_index == 0 then
        self.marshaled_state[0xb8 + 0x08 + 1] = 0xb0
        self.marshaled_state[0xb8 + 0x09 + 1] = 0xa9
    else
        self.marshaled_state[0xb8 + 0x08 + 1] = 0x88
        self.marshaled_state[0xb8 + 0x09 + 1] = 0xaa
    end
end

function Client:recv_marshaled_state()
    return self.marshaled_state
end

return Client
