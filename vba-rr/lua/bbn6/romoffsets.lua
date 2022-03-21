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
        battle_end__entry                                           = 0x08007ca0,
        commMenu_waitForFriend__call__commMenu_handleLinkCableInput = 0x08129f8a,
        commMenu_inBattle__call__commMenu_handleLinkCableInput      = 0x0812b5ca,
        battle_isRemote__ret                                        = 0x0803dd52,
        link_isRemote__ret                                          = 0x0803dd86,
    },

    _MEGAMAN6_GXX = {  -- TODO: figure out why ths is busted
        commMenu_handleLinkCableInput__entry                        = 0x0803eaae,
        battle_init__call__battle_copyInputData                     = 0x080078f8,
        battle_update__call__battle_copyInputData                   = 0x08007a64,
        battle_getSettingsForLink__ret                              = 0x0802d274,
        battle_settings_list1                                       = 0x080b25ee,
        battle_init_marshal__ret                                    = 0x0800b2ae,
        battle_turn_marshal__ret                                    = 0x0800b3cc,
        battle_updating__ret__go_to_custom_screen                   = 0x080093a4,
        battle_start__ret                                           = 0x080072fa,
        battle_end__entry                                           = 0x08007c96,
        commMenu_waitForFriend__call__commMenu_handleLinkCableInput = 0x0812bd6a,
        commMenu_inBattle__call__commMenu_handleLinkCableInput      = 0x0812d3aa,
        battle_isRemote__ret                                        = 0x0803dd1c,
        link_isRemote__ret                                          = 0x0803dd50,
    }
}

assert(offsets[title] ~= nil, "game not supported")
return offsets[title]
