package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "verana/x/pp/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgStartParticipantOP{}, "verana/x/pp/MsgStartParticipantOP")
	legacy.RegisterAminoMsg(cdc, &MsgRenewParticipantOP{}, "verana/x/pp/MsgRenewParticipantOP")
	legacy.RegisterAminoMsg(cdc, &MsgSetParticipantOPToValidated{}, "verana/x/pp/MsgSetPartOPValidated")
	legacy.RegisterAminoMsg(cdc, &MsgCancelParticipantOPLastRequest{}, "verana/x/pp/MsgCancelPartOPLastReq")
	legacy.RegisterAminoMsg(cdc, &MsgCreateRootParticipant{}, "verana/x/pp/MsgCreateRootParticipant")
	legacy.RegisterAminoMsg(cdc, &MsgSetParticipantEffectiveUntil{}, "verana/x/pp/MsgSetPartEffectiveUntil")
	legacy.RegisterAminoMsg(cdc, &MsgRevokeParticipant{}, "verana/x/pp/MsgRevokeParticipant")
	legacy.RegisterAminoMsg(cdc, &MsgCreateOrUpdateParticipantSession{}, "verana/x/pp/MsgCreateOrUpdatePartSess")
	legacy.RegisterAminoMsg(cdc, &MsgSlashParticipantTrustDeposit{}, "verana/x/pp/MsgSlashParticipantTD")
	legacy.RegisterAminoMsg(cdc, &MsgRepayParticipantSlashedTrustDeposit{}, "verana/x/pp/MsgRepayPartSlashedTD")
	legacy.RegisterAminoMsg(cdc, &MsgSelfCreateParticipant{}, "verana/x/pp/MsgSelfCreateParticipant")
	legacy.RegisterAminoMsg(cdc, &MsgTriggerResolver{}, "verana/x/pp/MsgTriggerResolver")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgStartParticipantOP{},
		&MsgRenewParticipantOP{},
		&MsgSetParticipantOPToValidated{},
		&MsgCancelParticipantOPLastRequest{},
		&MsgCreateRootParticipant{},
		&MsgSetParticipantEffectiveUntil{},
		&MsgRevokeParticipant{},
		&MsgCreateOrUpdateParticipantSession{},
		&MsgSlashParticipantTrustDeposit{},
		&MsgRepayParticipantSlashedTrustDeposit{},
		&MsgSelfCreateParticipant{},
		&MsgTriggerResolver{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
