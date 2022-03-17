# Notes

`0x020099cc`: `u8` Flips the screen around.

## Objects

`0x080032d4 + objectype * 0x10`: `u32` Pointer to start of objects (actual object starts `0x10` in, first 16 bytes are pointers to a linked list).

`0x080032d8 + objectype * 0x10`: `u32` Pointer to end of objects.

`0x080032dc + objectype * 0x10`: `u8` Pointer to size of objects.

    - Type 1: start = `0x0203a9a0`, end = `0x0203c4a0`, size = `0xd8`
    - Type 3: start = `0x0203cfd0`, end = `0x0203ead0`, size = `0xd8`
    - Type 4: start = `0x02036860`, end = `0x02038160`, size = `0xc8`
