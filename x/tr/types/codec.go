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
	legacy.RegisterAminoMsg(cdc, &MsgCreateTrustRegistry{}, "verana/x/tr/MsgCreateTrustRegistry")
	legacy.RegisterAminoMsg(cdc, &MsgAddGovernanceFrameworkDocument{}, "verana/x/tr/MsgAddGovFrameworkDoc")
	legacy.RegisterAminoMsg(cdc, &MsgIncreaseActiveGovernanceFrameworkVersion{}, "verana/x/tr/MsgIncreaseActiveGovFWVer")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateTrustRegistry{}, "verana/x/tr/MsgUpdateTrustRegistry")
	legacy.RegisterAminoMsg(cdc, &MsgArchiveTrustRegistry{}, "verana/x/tr/MsgArchiveTrustRegistry")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateTrustRegistry{},
		&MsgAddGovernanceFrameworkDocument{},
		&MsgIncreaseActiveGovernanceFrameworkVersion{},
		&MsgUpdateTrustRegistry{},
		&MsgArchiveTrustRegistry{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
