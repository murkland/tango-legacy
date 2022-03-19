local memory = require("./memory")

function set_p1_input(joyflags)
    memory.write_u16(0x02009488, joyflags)
end

function set_p2_input(joyflags)
    memory.write_u16(0x0200948a, joyflags)
end

function set_battle_rng(val)
    memory.write_u32(0x020013f0, val)
end

return {
    set_p1_input = set_p1_input,
    set_p2_input = set_p2_input,
    set_battle_rng = set_battle_rng,
}
