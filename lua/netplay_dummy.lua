local function table_copy(t)
    local u = { }
    for k, v in pairs(t) do u[k] = v end
    return setmetatable(u, getmetatable(t))
  end

local Client = {}
Client.__index = Client

function Client.new()
    local client = {
        init = nil,
        input = nil,
        turn = nil,
        input = 0xfc00,
    }
    setmetatable(client, Client)
    return client
end

function Client:give_input(tick, input)
    self.input = input
end

function Client:take_input()
    local input = self.input
    self.input = nil
    return input
end

function Client:give_init(init)
    self.init = init
end

function Client:take_init()
    local init = self.init
    self.init = nil
    return init
end

function Client:give_turn(turn)
    self.turn = turn
end

function Client:take_turn()
    local turn = self.turn
    self.turn = nil
    return turn
end

function Client:start(loop)
end

return Client
