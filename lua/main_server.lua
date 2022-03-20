local HOST = "localhost"
local PORT = 12345

local log = require("./log")

local entry = require("./entry")

local socket = require("socket")

log.info("this is the SERVER!")

local lis = assert(socket.bind(HOST, PORT))
local host, port = lis:getsockname()

log.info("listening on %s:%d", host, port)

local emulator = require("./platform/require")("emulator")

local sock
while true do
    local readable, writable, err = socket.select({lis}, {}, 0)
    if #readable > 0 then
        break
    end
    emulator.advance_frame()
end

local sock = assert(lis:accept())
local host, port = sock:getsockname()
log.info("received client on %s:%d", host, port)

entry(sock, 0)
