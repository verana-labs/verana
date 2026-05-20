package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgAddGovernanceFrameworkDocument{},
		&MsgIncreaseActiveGovernanceFrameworkVersion{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "verana/x/gf/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgAddGovernanceFrameworkDocument{}, "verana/x/gf/MsgAddGovernanceFrameworkDocument", nil)
	cdc.RegisterConcrete(&MsgIncreaseActiveGovernanceFrameworkVersion{}, "verana/x/gf/MsgIncreaseActiveGovernanceFrameworkVersion", nil)
}
