local DEBUG = 0
local INFO = 1
local WARN = 2
local ERROR = 3

local LEVEL_NAMES = {
    [DEBUG] = "DEBUG",
    [INFO] = "INFO",
    [WARN] = "WARN",
    [ERROR] = "ERROR",
}

function format(level, fmt, ...)
    return os.date() .. ": " .. LEVEL_NAMES[level] .. ": " .. string.format(fmt, unpack(arg))
end

function make_log_func(level)
    return function(fmt, ...)
        log(level, fmt, unpack(arg))
    end
end

function log(level, fmt, ...)
    print(format(level, fmt, unpack(arg)))
end

return {
    DEBUG = DEBUG,
    INFO = INFO,
    WARN = WARN,
    ERROR = ERROR,

    debug = make_log_func(DEBUG),
    info = make_log_func(INFO),
    warn = make_log_func(WARN),
    error = make_log_func(ERROR),
}
