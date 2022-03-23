package bn6

type Offsets struct {
	A_battle_init__call__battle_copyInputData                     uint32
	A_battle_update__call__battle_copyInputData                   uint32
	A_battle_getSettingsForLink__ret                              uint32
	A_battle_settings_list1                                       uint32
	A_battle_init_marshal__ret                                    uint32
	A_battle_turn_marshal__ret                                    uint32
	A_battle_updating__ret__go_to_custom_screen                   uint32
	A_battle_start__ret                                           uint32
	A_battle_end__entry                                           uint32
	A_battle_isRemote__tst                                        uint32
	A_link_isRemote__ret                                          uint32
	A_commMenu_handleLinkCableInput__entry                        uint32
	A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput uint32
	A_commMenu_inBattle__call__commMenu_handleLinkCableInput      uint32
}

var offsetsMap = map[string]Offsets{
	"MEGAMAN6_FXX": {
		A_battle_init__call__battle_copyInputData:                     0x08007902,
		A_battle_update__call__battle_copyInputData:                   0x08007a6e,
		A_battle_getSettingsForLink__ret:                              0x0802d27e,
		A_battle_settings_list1:                                       0x080b0d88,
		A_battle_init_marshal__ret:                                    0x0800b2b8,
		A_battle_turn_marshal__ret:                                    0x0800b3d6,
		A_battle_updating__ret__go_to_custom_screen:                   0x0800945a,
		A_battle_start__ret:                                           0x08007304,
		A_battle_end__entry:                                           0x08007ca0,
		A_battle_isRemote__tst:                                        0x0803dd52,
		A_link_isRemote__ret:                                          0x0803dd86,
		A_commMenu_handleLinkCableInput__entry:                        0x0803eae4,
		A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x08129f8a,
		A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x0812b5ca,
	},
	"MEGAMAN6_GXX": {
		A_battle_init__call__battle_copyInputData:                     0x08007902,
		A_battle_update__call__battle_copyInputData:                   0x08007a6e,
		A_battle_getSettingsForLink__ret:                              0x0802d27e,
		A_battle_settings_list1:                                       0x080b0d88,
		A_battle_init_marshal__ret:                                    0x0800b2b8,
		A_battle_turn_marshal__ret:                                    0x0800b3d6,
		A_battle_updating__ret__go_to_custom_screen:                   0x0800945a,
		A_battle_start__ret:                                           0x08007304,
		A_battle_end__entry:                                           0x08007ca0,
		A_battle_isRemote__tst:                                        0x0803dd26,
		A_link_isRemote__ret:                                          0x0803dd5a,
		A_commMenu_handleLinkCableInput__entry:                        0x0803eab8,
		A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x0812bd66,
		A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x0812d3a6,
	},
}

func OffsetsForGame(title string) (Offsets, bool) {
	offsets, ok := offsetsMap[title]
	return offsets, ok
}
