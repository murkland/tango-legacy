local rom = require("./rom")

local title = rom.get_title()
print("detected game: " .. title)

local offsets = {
    MEGAMAN6_FXX = {
        battle_handleLinkCableInput__call__battle_handleLinkSIO     = 0x0803eb04,
        battle_update__call__battle_copyInputData                   = 0x08007a6c,
        battle_custom_complete__ret                                 = 0x08026ddc,
        battle_updating__ret__go_to_custom_screen                   = 0x080093ae,
    }
}

assert(offsets[title] ~= nil, "game not supported")
return offsets[title]
