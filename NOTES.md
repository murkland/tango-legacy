# Notes

`0x020099cc`: `u8` Flips the screen around.

`0x020093a4`: `u8` Battle active

`0x080032d4 + objectype * 0x10`: `u32` Pointer to start of objects (actual object starts `0x10` in, first 16 bytes are pointers to a linked list).

`0x080032d8 + objectype * 0x10`: `u32` Pointer to end of objects.

`0x080032dc + objectype * 0x10`: `u8` Pointer to size of objects.

-   Type 1: start = `0x0203a9b0`, end = `0x0203c4a0`, size = `0xd8`
-   Type 3: start = `0x0203cfe0`, end = `0x0203ead0`, size = `0xd8`
-   Type 4: start = `0x02036870`, end = `0x02038160`, size = `0xc8`

`0x020349c0`: `u16` Player 1 chip index
`0x020349c2`: `u16` Player 1 chip 0
`0x020349c4`: `u16` Player 1 chip 1, etc.

`0x02034a10`: `u16` Player 2 chip index
`0x02034a12`: `u16` Player 2 chip 0
`0x02034a14`: `u16` Player 2 chip 1, etc.

`0x0203f4a4`: Custscreen chips, same format, gets copied to `0x020349c0` after cust screen exit
