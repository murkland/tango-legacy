package bn6

type EWRAMOffsets struct {
	A_PlayerInputDataArr        uint32
	A_BattleState               uint32
	A_Joypad                    uint32
	A_LocalMarshaledBattleState uint32
	A_PlayerMarshaledStateArr   uint32
	A_MenuControl               uint32
	A_Rng2                      uint32
}

type ROMOffsets struct {
	A_battle_init__call__battle_copyInputData                     uint32
	A_battle_update__call__battle_copyInputData                   uint32
	A_battle_copyInputData__entry                                 uint32
	A_battle_settings_list1                                       uint32
	A_battle_init_marshal__ret                                    uint32
	A_battle_turn_marshal__ret                                    uint32
	A_battle_updating__ret__go_to_custom_screen                   uint32
	A_battle_start__ret                                           uint32
	A_battle_end__entry                                           uint32
	A_battle_isP2__tst                                            uint32
	A_link_isP2__ret                                              uint32
	A_commMenu_handleLinkCableInput__entry                        uint32
	A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput uint32
	A_commMenu_waitForFriend__ret__cancel                         uint32
	A_commMenu_inBattle__call__commMenu_handleLinkCableInput      uint32
}

type Offsets struct {
	EWRAM EWRAMOffsets
	ROM   ROMOffsets
}

var bn6EWRAMOffsets = EWRAMOffsets{
	A_PlayerInputDataArr:        0x02036820,
	A_BattleState:               0x02034880,
	A_Joypad:                    0x0200a270,
	A_LocalMarshaledBattleState: 0x0203cbe0,
	A_PlayerMarshaledStateArr:   0x0203f4a0,
	A_MenuControl:               0x02009a30,
	A_Rng2:                      0x020013f0,
}

var offsetsMap = map[string]Offsets{
	"MEGAMAN6_FXX": {
		EWRAM: bn6EWRAMOffsets,
		ROM: ROMOffsets{
			A_battle_init__call__battle_copyInputData:                     0x08007902,
			A_battle_update__call__battle_copyInputData:                   0x08007a6e,
			A_battle_copyInputData__entry:                                 0x0801feee,
			A_battle_settings_list1:                                       0x080b0d88,
			A_battle_init_marshal__ret:                                    0x0800b2b8,
			A_battle_turn_marshal__ret:                                    0x0800b3d6,
			A_battle_updating__ret__go_to_custom_screen:                   0x080093ae,
			A_battle_start__ret:                                           0x08007304,
			A_battle_end__entry:                                           0x08007ca0,
			A_battle_isP2__tst:                                            0x0803dd52,
			A_link_isP2__ret:                                              0x0803dd86,
			A_commMenu_handleLinkCableInput__entry:                        0x0803eae4,
			A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x08129f8a,
			A_commMenu_waitForFriend__ret__cancel:                         0x08129fa4,
			A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x0812b5ca,
		},
	},
	"MEGAMAN6_GXX": {
		EWRAM: bn6EWRAMOffsets,
		ROM: ROMOffsets{
			A_battle_init__call__battle_copyInputData:                     0x08007902,
			A_battle_update__call__battle_copyInputData:                   0x08007a6e,
			A_battle_copyInputData__entry:                                 0x0801feee,
			A_battle_settings_list1:                                       0x080b0d88,
			A_battle_init_marshal__ret:                                    0x0800b2b8,
			A_battle_turn_marshal__ret:                                    0x0800b3d6,
			A_battle_updating__ret__go_to_custom_screen:                   0x080093ae,
			A_battle_start__ret:                                           0x08007304,
			A_battle_end__entry:                                           0x08007ca0,
			A_battle_isP2__tst:                                            0x0803dd26,
			A_link_isP2__ret:                                              0x0803dd5a,
			A_commMenu_handleLinkCableInput__entry:                        0x0803eab8,
			A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x0812bd66,
			A_commMenu_waitForFriend__ret__cancel:                         0x0812bd80,
			A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x0812d3a6,
		},
	},
}
