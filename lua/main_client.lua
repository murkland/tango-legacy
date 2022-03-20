local HOST = "localhost"
local PORT = 12345

local log = require("./log")

local entry = require("./entry")

log.info("this is the CLIENT!")

local sock = assert(socket.connect(HOST, PORT))
local host, port = sock:getpeername()
log.info("connected to %s:%d", host, port)
sock:setoption("tcp-nodelay", true)

entry(sock, 1)
