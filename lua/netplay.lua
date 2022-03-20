local emulator = require("./platform/require")("emulator")

local coroutine = require("coroutine")
local Cosocket = require("./aio/cosocket")
local coutil = require("./aio/coutil")
local log = require("./log")
local struct = require("struct")

local PACKET_TYPE_INIT = '\0'
local PACKET_TYPE_INPUT = '\1'
local PACKET_TYPE_TURN = '\2'

local Client = {}
Client.__index = Client

function Client.new(sock)
    local self = {
        sock = Cosocket.new(sock),

        is_in_battle = false,

        local_init = nil,
        remote_init = nil,

        local_input = nil,
        remote_input = nil,

        local_turn = nil,
        remote_turn = nil,
    }
    setmetatable(self, Client)
    return self
end

function Client:give_input(tick, joyflags)
    self.local_input = joyflags
end

function Client:take_input()
    local input = self.remote_input
    self.remote_input = nil
    return input
end

function Client:give_init(init)
    self.local_init = init
    self.is_in_battle = true
end

function Client:take_init()
    local init = self.remote_init
    self.remote_init = nil
    return init
end

function Client:give_turn(turn)
    self.local_turn = turn
end

function Client:take_turn()
    local turn = self.remote_turn
    self.remote_turn = nil
    return turn
end

function u8table_to_string(t)
    local s = {}
    for i, c in ipairs(t) do
        s[i] = string.char(c)
    end
    return table.concat(s)
end

function string_to_u8table(s)
    local t = {}
    for c in s:gmatch('.') do
        t[#t+1] = c:byte(1)
    end
    return t
end

function Client:run(loop)
    while true do
        if self.is_in_battle then
            if self.local_init ~= nil then
                local init = self.local_init
                self.sock:send(loop, PACKET_TYPE_INIT .. u8table_to_string(init))
                self.local_init = nil
            end

            if self.local_input ~= nil then
                local input = self.local_input
                self.sock:send(loop, PACKET_TYPE_INPUT .. struct.write("w", input))
                self.local_input = nil
            end

            if self.local_turn ~= nil then
                local turn = self.local_turn
                self.sock:send(loop, PACKET_TYPE_TURN .. u8table_to_string(turn))
                self.local_turn = nil
            end

            if self.sock:readable() then
                local op = self.sock:receive(loop, 1)
                if op == PACKET_TYPE_INIT then
                    self.remote_init = string_to_u8table(self.sock:receive(loop, 0x100))
                elseif op == PACKET_TYPE_INPUT then
                    local l = self.sock:receive(loop, 4)
                    self.remote_input = struct.read(l, "w")[1]
                elseif op == PACKET_TYPE_TURN then
                    self.remote_turn = string_to_u8table(self.sock:receive(loop, 0x100))
                end
            end
        end
        coutil.yield(loop)
    end
end

function Client:start(loop)
    loop:add_callback(coroutine.wrap(function ()
        self:run(loop)
    end))
end

return Client
