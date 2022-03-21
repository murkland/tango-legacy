local HOST = "localhost"
local PORT = 12345

local log = require("./log")

local entry = require("./entry")

log.info("this is the DUMMY! unless you know what you're doing, don't use this!")

local Client = {}
Client.__index = Client

function Client.new()
    local client = {
        init = nil,
        input = nil,
        turn = nil,
        input = {tick = 0, joyflags = 0xfc00},
    }
    setmetatable(client, Client)
    return client
end

function Client:give_input(tick, joyflags)
    self.input = {tick = tick, joyflags = joyflags}
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

entry(Client, sock, 1)
