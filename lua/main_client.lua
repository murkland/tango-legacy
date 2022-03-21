local HOST = "localhost"
local PORT = 12345

local log = require("./log")

local hijack = require("./hijack")

log.info("this is the CLIENT!")

local sock = assert(socket.connect(HOST, PORT))
local host, port = sock:getpeername()
log.info("connected to %s:%d", host, port)
sock:setoption("tcp-nodelay", true)

local Client = require("./netplay")

hijack(Client, sock, 1)
