local HOST = "localhost"
local PORT = 12345

local log = require("./log")

local entry = require("./entry")

log.info("this is the DUMMY! unless you know what you're doing, don't use this!")
local Client = require("./netplay_dummy")

entry(Client, sock, 1)
