local HOST = "localhost"
local PORT = 12345

local log = require("./log")

local entry = require("./entry")

log.info("this is the CLIENT!")

local sock = socket.connect(HOST, PORT)
local host, port = sock:getpeername()

log.info("connected to %s:%d", host, port)

entry(sock, 1)
