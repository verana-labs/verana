package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgAddDID{},
		&MsgRenewDID{},
		&MsgRemoveDID{},
		&MsgTouchDID{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgAddDID{}, "/dtr/v1/dd/add-did")
	legacy.RegisterAminoMsg(cdc, &MsgRenewDID{}, "/dtr/v1/dd/renew-did")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveDID{}, "/dtr/v1/dd/remove-did")
	legacy.RegisterAminoMsg(cdc, &MsgTouchDID{}, "/dtr/v1/dd/touch-did")
}
