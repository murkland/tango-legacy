local memory = require("./platform/require")("memory")

local function u8arr_to_string(arr)
    local s = ""
    for _, c in ipairs(arr) do
        if c == 0 then break end
        s = s .. string.char(c)
    end
    return s
end

local function get_id()
    return u8arr_to_string(memory.read_range(0x080000ac, 4))
end

local function get_title()
    return u8arr_to_string(memory.read_range(0x080000a0, 12))
end

return {
    get_id = get_id,
    get_title = get_title,
}
