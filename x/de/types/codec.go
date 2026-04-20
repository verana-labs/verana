package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "verana/x/de/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgGrantOperatorAuthorization{}, "verana/x/de/MsgGrantOpAuthorization")
	legacy.RegisterAminoMsg(cdc, &MsgRevokeOperatorAuthorization{}, "verana/x/de/MsgRevokeOpAuthorization")
}

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgGrantOperatorAuthorization{},
		&MsgRevokeOperatorAuthorization{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
