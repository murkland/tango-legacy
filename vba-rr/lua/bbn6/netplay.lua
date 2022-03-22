local emulator = require("bbn6.platform.require")("emulator")

local coroutine = require("coroutine")

local socket = require("socket")
local log = require("bbn6.log")
local Deque = require("bbn6.deque")
local struct = require("bbn6.struct")

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

local PACKET_TYPE_INIT = '\0'
local PACKET_TYPE_INPUT = '\1'
local PACKET_TYPE_TURN = '\2'

local Client = {}
Client.__index = Client

function Client.new(sock, delay)
    local local_input_queue = Deque.new()
    local remote_input_queue = Deque.new()

    if delay == nil then
        -- random guess
        delay = 3
    end
    for i = 1, delay do
        local_input_queue:pushright({tick = i - delay - 1, joyflags = 0xfc00, custom_state = 0})
        remote_input_queue:pushright({tick = i - delay - 1, joyflags = 0xfc00, custom_state = 0})
    end

    if max_delay == nil then
        -- random guess
        max_delay = 6
    end

    local self = {
        sock = sock,

        delay = delay,

        local_turn = nil,
        remote_turn = nil,

        local_input_queue = local_input_queue,
        remote_input_queue = remote_input_queue,
    }
    setmetatable(self, Client)
    return self
end

function Client:queue_local_input(tick, joyflags, custom_state)
    assert(self.sock:send(PACKET_TYPE_INPUT .. struct.write("dwb", tick, joyflags, custom_state)))
    self.local_input_queue:pushright({tick = tick, joyflags = joyflags, custom_state = custom_state})
    return true
end

function Client:dequeue_inputs()
    while self.remote_input_queue:len() < self.delay do
        local op = assert(self.sock:receive(1))
        assert(#op == 1)

        if op == PACKET_TYPE_TURN then
            local tick = struct.read(assert(self.sock:receive(4)), "d")[1]
            self.remote_turn = {tick = tick, marshaled = string_to_u8table(assert(self.sock:receive(0x100)))}
        elseif op == PACKET_TYPE_INPUT then
            local l = struct.read(assert(self.sock:receive(7), "dwb"))
            local remote_input = {tick = l[1], joyflags = l[2], custom_state = l[3]}
            self.remote_input_queue:pushright(remote_input)
        else
            error("unexpected packet type: " .. op:byte(1))
        end
    end

    local local_ = self.local_input_queue:popleft()
    local remote = self.remote_input_queue:popleft()

    assert(local_.tick == remote.tick)
    local ret = {
        tick = local_.tick,
        local_ = local_,
        remote = remote,
    }

    if self.local_turn ~= nil and self.local_turn.tick + 1 == local_.tick then
        ret.local_turn = self.local_turn.marshaled
        self.local_turn = nil
    end

    if self.remote_turn ~= nil and self.remote_turn.tick + 1 == remote.tick then
        ret.remote_turn = self.remote_turn.marshaled
        self.remote_turn = nil
    end

    return ret
end

function Client:give_init(init)
    assert(self.sock:send(PACKET_TYPE_INIT .. u8table_to_string(init)))
end

function Client:take_init()
    local readable, _, err = socket.select({self.sock}, {}, 0)
    assert(err == nil or err == "timeout", err)
    if #readable == 0 then
        return nil
    end
    local op = self.sock:receive(1)
    assert(op == PACKET_TYPE_INIT, "unexpected packet type: " .. op:byte(1))
    return string_to_u8table(assert(self.sock:receive(0x100)))
end

function Client:queue_local_turn(tick, marshaled)
    self.local_turn = {tick = tick, marshaled = marshaled}
    assert(self.sock:send(PACKET_TYPE_TURN .. struct.write("d", tick) .. u8table_to_string(marshaled)))
end

return Client
