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
		&MsgCreateCorporation{},
		&MsgUpdateCorporation{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "verana/x/co/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgCreateCorporation{}, "verana/x/co/MsgCreateCorporation", nil)
	cdc.RegisterConcrete(&MsgUpdateCorporation{}, "verana/x/co/MsgUpdateCorporation", nil)
}
