-- struct.lua
-- Tiny binary writer and reader in Lua.

-- version:      1.3
-- author:       Martin 'Halt' Cohen
-- url:          https://twitter.com/martin_cohen
-- license:      MIT
-- changes:      at the end of this file
-- dependencies:
    -- http://bitop.luajit.org

-- example:
--[[
    local struct = require "struct"
    local f = io.open("image.png", "rb")
    print("png size", unpack(struct.read(f:read("*all"), ".DD", 0x10)))
    f:close()
--]]

local bit = require("bit")
local band = bit.band
local bor = bit.bor
local rshift = bit.rshift
local lshift = bit.lshift
local tohex = bit.tohex
local bswap4 = bit.bswap
local push = table.insert
local schar = string.char
local sbyte = string.byte

local struct = {}

local function bswap2(x, n)
    return rshift(bswap4(x), 16)
end

local function wnum(bytes, n, c)
    for i = 1, c do
        push(bytes, schar(band(n, 0xFF)))
        n = rshift(n, 8)
    end
end

-- b    uint8
-- w    uint16
-- d    uint32
-- W    uint16 be
-- D    uint16 be
-- s    string
-- S    write uint32 length and then write string
-- z*   zero-padded or truncated string, length is defined by arg before string argument
-- z<n> zero-padded or truncated string, length is defined by `<n>`; for example `z8`
function struct.write(format, ...)
    local bytes = {}

    local a = 1 -- '...' iterator
    local i = 1 -- 'format' iterator
    while i <= #format do
        local v = select(a, ...)
        local f = format:sub(i, i)

            if f == "b" then push(bytes, schar(v))
        elseif f == "B" then push(bytes, schar(v))
        elseif f == "w" then wnum(bytes, v, 2)
        elseif f == "W" then wnum(bytes, bswap2(v, 2), 2)
        elseif f == "d" then wnum(bytes, v, 4)
        elseif f == "D" then wnum(bytes, bswap4(v), 4)
        elseif f == "s" then push(bytes, tostring(v))
        elseif f == "S" then
            wnum(bytes, #v, 4)
            push(bytes, tostring(v))
        elseif f == "z" then
            local n
            if format:sub(i + 1, i + 1) == "*" then
                i = i + 1
                a = a + 1
                n = v
                v = select(a, ...)
            else
                n = format:match("[0-9]+", i + 1)
                if not n then error(string.format("number or * expected after 'z' at %d", i)) end
                i = i + #n
                n = tonumber(n)
            end
            push(bytes, v:sub(1, n))
            if n > #v then push(bytes, string.rep("\0", n - #v)) end
        else
            error(string.format("unknown format '%s'", f))
        end
        i = i + 1
        a = a + 1
    end

    return table.concat(bytes)
end

local function rnum(bytes, k, n)
    local args = { sbyte(bytes, k, k + n - 1) }
    local v = 0
    for i = 1, #args do
        v = bor(v, lshift(args[i], (i - 1) * 8))
    end
    return v
end

-- b    uint8
-- w    uint16
-- d    uint32
-- W    uint16 be
-- D    uint16 be
-- .    skip <n> bytes, (n is denoted by extra arg in order)
-- s    read <n> bytes as string
-- S    read uint32 for length and then read string of that length
-- z*   zero-padded string, length is defined by arg
-- z<n> zero-padded string, length is defined by `<n>`; for example `z8`
function struct.read(bytes, format, ...)
    assert(type(bytes) == "string")
    assert(type(format) == "string")

    local i = 1 -- 'format' iterator
    local a = 1 -- '...' iterator
    local k = 1 -- 'bytes' iterator
    local values = {}
    while i <= #format do
        local f = format:sub(i, i)
            if f == "b" then push(values, sbyte(bytes, k)); k = k + 1
        elseif f == "w" then push(values, rnum(bytes, k, 2)); k = k + 2
        elseif f == "d" then push(values, rnum(bytes, k, 4)); k = k + 4
        elseif f == "B" then push(values, sbyte(bytes, k)); k = k + 1
        elseif f == "W" then push(values, bswap2(rnum(bytes, k, 2))); k = k + 2
        elseif f == "D" then push(values, bswap4(rnum(bytes, k, 4))); k = k + 4
        elseif f == "." then k = k + select(a, ...); a = a + 1
        elseif f == "s" then
            local n = select(a, ...)
            push(values, bytes:sub(k, k + n - 1))
            k = k + n
            a = a + 1
        elseif f == "S" then
            local n = rnum(bytes, k, 4)
            k = k + 4
            push(values, bytes:sub(k, k + n - 1))
            k = k + n
        elseif f == "z" then
            local n
            if format:sub(i + 1, i + 1) == "*" then
                -- z*
                n = select(a, ...)
                i = i + 1
                a = a + 1
            else
                -- z<n>
                n = format:match("[0-9]+", i + 1)
                if not n then error(string.format("number or * expected after 'z' at %d", i)) end
                i = i + #n
                n = tonumber(n)
            end

            -- trim zeroes
            local s = bytes:sub(k, k + n - 1)
            for i = #s, 0, -1 do
                if i == 0 then s = "" break
                elseif sbyte(s, i) ~= 0 then s = s:sub(1, i) break end
            end
            push(values, s)
            k = k + n
        else
            error(string.format("unknown format '%s'", f))
        end
        i = i + 1
    end
    return values
end

-- reads binary string and returns string with hexadecimal numbers
-- `hexdump("hello") -> "68 65 6c 6c 6f"`
function struct.hexdump(x)
    assert(type(x) == "string")

    local t = {}
    for _, i in ipairs{ sbyte(x, 1, #x) } do
        push(t, tohex(i, 2))
    end
    return table.concat(t, " ")
end

-- reads string with hexadecimal numbers, and returns it as binary
-- `hexload("68 65 6c 6c 6f") -> "hello"
function struct.hexload(x)
    assert(type(x) == "string")

    local bytes = {}
    local i = 1
    while true do
        local m = x:match("([0-9a-fA-F][0-9a-fA-F]%s?)", i)
        if not m then break end
        i = i + #m
        push(bytes, schar(tonumber(m, 16)))
    end

    return table.concat(bytes)
end

return struct

-- v1.3
    -- added 'hexload'
    -- added 'z*' format for read and write
    -- added 'z<n>' format for read and write
-- v1.2
    -- error on bad format in 'write'
-- v1.1
    -- added 'S' to write/read length and a string
-- v1.0
    -- initial release
