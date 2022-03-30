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
	A_battle_runUnpausedStep__cmp__retval                         uint32
	A_battle_copyInputData__entry                                 uint32
	A_battle_init_marshal__ret                                    uint32
	A_battle_turn_marshal__ret                                    uint32
	A_battle_updating__ret__go_to_custom_screen                   uint32
	A_battle_start__ret                                           uint32
	A_battle_end__entry                                           uint32
	A_battle_isP2__tst                                            uint32
	A_link_isP2__ret                                              uint32
	A_commMenu_initBattle__entry                                  uint32
	A_commMenu_handleLinkCableInput__entry                        uint32
	A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput uint32
	A_commMenu_waitForFriend__ret__cancel                         uint32
	A_commMenu_inBattle__call__commMenu_handleLinkCableInput      uint32
	A_commMenu_endBattle__entry                                   uint32
}

type Offsets struct {
	EWRAM EWRAMOffsets
	ROM   ROMOffsets
}

var ewramOffsets = EWRAMOffsets{
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
		EWRAM: ewramOffsets,
		ROM: ROMOffsets{
			A_battle_init__call__battle_copyInputData:                     0x08007902,
			A_battle_update__call__battle_copyInputData:                   0x08007a6e,
			A_battle_runUnpausedStep__cmp__retval:                         0x08008102,
			A_battle_copyInputData__entry:                                 0x0801feee,
			A_battle_init_marshal__ret:                                    0x0800b2b8,
			A_battle_turn_marshal__ret:                                    0x0800b3d6,
			A_battle_updating__ret__go_to_custom_screen:                   0x080093ae,
			A_battle_start__ret:                                           0x08007304,
			A_battle_end__entry:                                           0x08007ca0,
			A_battle_isP2__tst:                                            0x0803dd52,
			A_link_isP2__ret:                                              0x0803dd86,
			A_commMenu_initBattle__entry:                                  0x0812b608,
			A_commMenu_handleLinkCableInput__entry:                        0x0803eae4,
			A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x08129f8a,
			A_commMenu_waitForFriend__ret__cancel:                         0x08129fa4,
			A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x0812b5ca,
			A_commMenu_endBattle__entry:                                   0x0812b708,
		},
	},
	"MEGAMAN6_GXX": {
		EWRAM: ewramOffsets,
		ROM: ROMOffsets{
			A_battle_init__call__battle_copyInputData:                     0x08007902,
			A_battle_update__call__battle_copyInputData:                   0x08007a6e,
			A_battle_runUnpausedStep__cmp__retval:                         0x08008102,
			A_battle_copyInputData__entry:                                 0x0801feee,
			A_battle_init_marshal__ret:                                    0x0800b2b8,
			A_battle_turn_marshal__ret:                                    0x0800b3d6,
			A_battle_updating__ret__go_to_custom_screen:                   0x080093ae,
			A_battle_start__ret:                                           0x08007304,
			A_battle_end__entry:                                           0x08007ca0,
			A_battle_isP2__tst:                                            0x0803dd26,
			A_link_isP2__ret:                                              0x0803dd5a,
			A_commMenu_initBattle__entry:                                  0x0812d3e4,
			A_commMenu_handleLinkCableInput__entry:                        0x0803eab8,
			A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x0812bd66,
			A_commMenu_waitForFriend__ret__cancel:                         0x0812bd80,
			A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x0812d3a6,
			A_commMenu_endBattle__entry:                                   0x0812d4e4,
		},
	},
	"ROCKEXE6_RXX": {
		EWRAM: ewramOffsets,
		ROM: ROMOffsets{
			A_battle_init__call__battle_copyInputData:                     0x080078ee,
			A_battle_update__call__battle_copyInputData:                   0x08007a6a,
			A_battle_runUnpausedStep__cmp__retval:                         0x0800811a,
			A_battle_copyInputData__entry:                                 0x08020302,
			A_battle_init_marshal__ret:                                    0x0800b8a0,
			A_battle_turn_marshal__ret:                                    0x0800b9be,
			A_battle_updating__ret__go_to_custom_screen:                   0x0800957e,
			A_battle_start__ret:                                           0x080072f8,
			A_battle_end__entry:                                           0x08007c9c,
			A_battle_isP2__tst:                                            0x0803ed96,
			A_link_isP2__ret:                                              0x0803edca,
			A_commMenu_initBattle__entry:                                  0x08134008,
			A_commMenu_handleLinkCableInput__entry:                        0x0803fb28,
			A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x0813299e,
			A_commMenu_waitForFriend__ret__cancel:                         0x081329b8,
			A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x08133fca,
			A_commMenu_endBattle__entry:                                   0x08134108,
		},
	},
	"ROCKEXE6_GXX": {
		EWRAM: ewramOffsets,
		ROM: ROMOffsets{
			A_battle_init__call__battle_copyInputData:                     0x080078ee,
			A_battle_update__call__battle_copyInputData:                   0x08007a6a,
			A_battle_runUnpausedStep__cmp__retval:                         0x0800811a,
			A_battle_copyInputData__entry:                                 0x08020302,
			A_battle_init_marshal__ret:                                    0x0800b8a0,
			A_battle_turn_marshal__ret:                                    0x0800b9be,
			A_battle_updating__ret__go_to_custom_screen:                   0x0800957e,
			A_battle_start__ret:                                           0x080072f8,
			A_battle_end__entry:                                           0x08007c9c,
			A_battle_isP2__tst:                                            0x0803ed6a,
			A_link_isP2__ret:                                              0x0803ed9e,
			A_commMenu_initBattle__entry:                                  0x08135dd0,
			A_commMenu_handleLinkCableInput__entry:                        0x0803fafc,
			A_commMenu_waitForFriend__call__commMenu_handleLinkCableInput: 0x08134766,
			A_commMenu_waitForFriend__ret__cancel:                         0x08134780,
			A_commMenu_inBattle__call__commMenu_handleLinkCableInput:      0x08135d92,
			A_commMenu_endBattle__entry:                                   0x08135ed0,
		},
	},
}
