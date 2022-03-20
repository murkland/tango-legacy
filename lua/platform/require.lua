local function _get_platform()
    if vba ~= nil then
        return "vba"
    end

    if bizstring ~= nil then
        return "bizhawk"
    end
end

local _platform = _get_platform()

print("detected emulator platform: " .. _platform)

local _require = function(path)
    return require("./platform/" .. path .. "_" .. _platform)
end

return _require
