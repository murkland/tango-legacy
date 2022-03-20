local DEBUG = 0
local INFO = 1
local WARN = 2
local ERROR = 3
local FATAL = 4

local LEVEL_NAMES = {
    [DEBUG] = "DEBUG",
    [INFO] = "INFO",
    [WARN] = "WARN",
    [ERROR] = "ERROR",
    [FATAL] = "FATAL",
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
    assert(level ~= FATAL)
end

return {
    DEBUG = DEBUG,
    INFO = INFO,
    WARN = WARN,
    ERROR = ERROR,
    FATAL = FATAL,

    debug = make_log_func(DEBUG),
    info = make_log_func(INFO),
    warn = make_log_func(WARN),
    error = make_log_func(ERROR),
    fatal = make_log_func(FATAL),
}
