local function table_copy(t)
    local u = { }
    for k, v in pairs(t) do u[k] = v end
    return setmetatable(u, getmetatable(t))
  end

local Client = {}
Client.__index = Client

function Client.new(player_index)
    local client = {
        init = nil,
        input = nil,
        turn = nil,
        player_index = player_index,
        joyflags = 0xfc00,
    }
    setmetatable(client, Client)
    return client
end

function Client:send_input(joyflags)
    self.joyflags = joyflags
end

function Client:take_input()
    local joyflags = self.joyflags
    self.joyflags = nil
    return joyflags
end

function Client:send_init(init)
    self.init = init
end

function Client:take_init()
    local init = self.init
    self.init = nil
    return init
end

function Client:send_turn(turn)
    self.turn = turn
end

function Client:take_turn()
    local turn = self.turn
    self.turn = nil
    return turn
end

return Client
