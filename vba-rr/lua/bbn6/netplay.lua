local emulator = require("bbn6.platform.require")("emulator")

local coroutine = require("coroutine")

local Cosocket = require("bbn6.aio.cosocket")
local coutil = require("bbn6.aio.coutil")
local log = require("bbn6.log")
local input = require("bbn6.input")
local Deque = require("bbn6.deque")
local struct = require("bbn6.struct")

local PACKET_TYPE_INIT = '\0'
local PACKET_TYPE_INPUT = '\1'
local PACKET_TYPE_TURN = '\2'

local Client = {}
Client.__index = Client

function Client.new(sock, min_delay, max_delay)
    local local_input_queue = Deque.new()
    local remote_input_queue = Deque.new()

    if min_delay == nil then
        -- random guess
        min_delay = 3
        for i = 1, min_delay do
            local_input_queue:pushright({tick = i - min_delay - 1, joyflags = input.Joyflag.DEFAULT, custom_state = 0})
            remote_input_queue:pushright({tick = i - min_delay - 1, joyflags = input.Joyflag.DEFAULT, custom_state = 0})
        end
    end

    if max_delay == nil then
        -- random guess
        max_delay = 6
    end

    local self = {
        min_delay = min_delay,
        max_delay = max_delay,

        sock = Cosocket.new(sock),

        is_in_battle = false,

        local_init = nil,
        remote_init = nil,

        pending_local_input_queue = Deque.new(),
        local_input_queue = local_input_queue,
        remote_input_queue = remote_input_queue,

        local_turn = nil,
        remote_turn = nil,
    }
    setmetatable(self, Client)
    return self
end

function Client:queue_local_input(tick, joyflags, custom_state)
    if self.pending_local_input_queue:len() >= self.max_delay then
        return false
    end
    self.pending_local_input_queue:pushright({tick = tick, joyflags = joyflags, custom_state = custom_state})
    return true
end

function Client:dequeue_inputs()
    if self.local_input_queue:len() < self.min_delay then
        return nil
    end

    if self.remote_input_queue:len() == 0 or self.local_input_queue:len() == 0 then
        return nil
    end

    local local_ = self.local_input_queue:popleft()
    local remote = self.remote_input_queue:popleft()
    assert(local_.ticks == remote.ticks)
    return {
        local_ = local_,
        remote = remote,
    }
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
                assert(self.sock:send(loop, PACKET_TYPE_INIT .. u8table_to_string(init)))
                self.local_init = nil
            end

            while self.pending_local_input_queue:len() > 0 do
                local input = self.pending_local_input_queue:popleft()
                assert(self.sock:send(loop, PACKET_TYPE_INPUT .. struct.write("dwb", input.tick, input.joyflags, input.custom_state)))
                self.local_input_queue:pushright(input)
            end

            if self.local_turn ~= nil then
                local turn = self.local_turn
                assert(self.sock:send(loop, PACKET_TYPE_TURN .. u8table_to_string(turn)))
                self.local_turn = nil
            end

            if self.sock:readable() then
                local op = assert(self.sock:receive(loop, 1))
                assert(#op == 1)
                if op == PACKET_TYPE_INIT then
                    self.remote_init = string_to_u8table(assert(self.sock:receive(loop, 0x100)))
                elseif op == PACKET_TYPE_INPUT then
                    local l = struct.read(assert(self.sock:receive(loop, 7), "dwb"))
                    self.remote_input_queue:pushright({tick = l[1], joyflags = l[2], custom_state = l[3]})
                elseif op == PACKET_TYPE_TURN then
                    self.remote_turn = string_to_u8table(assert(self.sock:receive(loop, 0x100)))
                end
            end
        end
        loop:add_callback(function()
            emulator.advance_frame()
        end)
        coutil.yield(loop)
    end
end

function Client:start(loop)
    loop:add_callback(coroutine.wrap(function ()
        self:run(loop)
    end))
end

return Client
