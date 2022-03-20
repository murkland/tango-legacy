local coroutine = require("coroutine")
local Cosocket = require("./aio/cosocket")
local coutil = require("./aio/coutil")
local struct = require("struct")

local PACKET_TYPE_INIT = 0
local PACKET_TYPE_INPUT = 1
local PACKET_TYPE_TURN = 2

local Client = {}
Client.__index = Client

function Client.new(sock)
    local self = {sock = Cosocket.new(sock)}
    setmetatable(self, Client)
    return self
end

function Client:send_input(tick, joyflags)
    self.sock.send(PACKET_TYPE_INPUT)
end

function Client:take_input()
end

function Client:send_init(init)
    self.sock.send(PACKET_TYPE_INIT)
end

function Client:take_init()
end

function Client:send_turn(turn)
    self.sock.send(PACKET_TYPE_TURN)
end

function Client:take_turn()
end

function Client:run(loop)
    while true do
        coutil.yield(loop)
    end
end

function Client:run_on_loop(loop)
    loop:add_callback(coroutine.wrap(function () self:run(loop) end))
end

return Client
