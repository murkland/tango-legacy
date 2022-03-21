local log = require("bbn6.log")
local rom = require("bbn6.rom")

local title = rom.get_title()
log.info("detected game: " .. title)

local offsets = {
    MEGAMAN6_FXX = {
        commMenu_handleLinkCableInput__entry                        = 0x0803eae4,
        battle_init__call__battle_copyInputData                     = 0x08007902,
        battle_update__call__battle_copyInputData                   = 0x08007a6e,
        battle_getSettingsForLink__ret                              = 0x0802d27e,
        battle_settings_list1                                       = 0x080b0d88,
        battle_init_marshal__ret                                    = 0x0800b2b8,
        battle_turn_marshal__ret                                    = 0x0800b3d6,
        battle_updating__ret__go_to_custom_screen                   = 0x080093ae,
        battle_start__ret                                           = 0x08007304,
        commMenu_waitForFriend__call__commMenu_handleLinkCableInput = 0x08129f8a,
        commMenu_inBattle__call__commMenu_handleLinkCableInput      = 0x0812b5ca,
        battle_isRemote__ret                                        = 0x0803dd52,
        link_isRemote__ret                                          = 0x0803dd86,
    }
}

assert(offsets[title] ~= nil, "game not supported")
return offsets[title]
