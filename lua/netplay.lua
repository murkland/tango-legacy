local socket = require("socket")
local struct = require("struct")

local PACKET_TYPE_INIT = 0
local PACKET_TYPE_INPUT = 1
local PACKET_TYPE_TURN = 2

local Client = {}
Client.__index = Client

function Client.new(addr, port)
    local sock = assert(socket.connect(addr, port))
    local client = {sock = sock}
    setmetatable(client, Client)
    return client
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

function Client:run_on_loop(loop)
end

return Client
