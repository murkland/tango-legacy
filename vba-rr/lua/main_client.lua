local HOST = "localhost"
local PORT = 5738

local log = require("bbn6.log")

local hijack = require("bbn6.hijack")

log.info("this is the CLIENT!")

local sock = assert(socket.connect(HOST, PORT))
local host, port = sock:getpeername()
log.info("connected to %s:%d", host, port)
sock:setoption("tcp-nodelay", true)

local Client = require("bbn6.netplay")

hijack(Client, sock, 1)
