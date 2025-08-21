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
	legacy.RegisterAminoMsg(cdc, &MsgAddDID{}, "/dtr/v1/dd/add-did")
	legacy.RegisterAminoMsg(cdc, &MsgRenewDID{}, "/dtr/v1/dd/renew-did")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveDID{}, "/dtr/v1/dd/remove-did")
	legacy.RegisterAminoMsg(cdc, &MsgTouchDID{}, "/dtr/v1/dd/touch-did")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgAddDID{},
		&MsgRenewDID{},
		&MsgRemoveDID{},
		&MsgTouchDID{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
