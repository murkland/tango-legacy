return {
    read_u8 = memory.readbyteunsigned,
    read_s8 = memory.readbyte,
    read_u16 = memory.readshortunsigned,
    read_s16 = memory.readshort,
    read_u32 = memory.readlongunsigned,
    read_s32 = memory.readlong,
    read_range = memory.readbyterange,
    read_reg = memory.getregister,

    write_u8 = memory.writebyte,
    write_s8 = memory.writebyte,
    write_u16 = memory.writeshort,
    write_s16 = memory.writeshort,
    write_u32 = memory.writelong,
    write_s32 = memory.writelong,
    write_range = function (offset, vs)
        for i, v in ipairs(vs) do
            memory.writebyte(offset + i - 1, v)
        end
    end,
    write_reg = memory.setregister,

    on_exec = memory.registerexec,
    on_write = memory.registerwrite,
}
