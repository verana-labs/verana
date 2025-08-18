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
		&MsgCreateCredentialSchema{},
		&MsgUpdateCredentialSchema{},
		&MsgArchiveCredentialSchema{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgCreateCredentialSchema{}, "/vpr/v1/cs/create-credential-schema")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateCredentialSchema{}, "/vpr/v1/cs/update-credential-schema")
	legacy.RegisterAminoMsg(cdc, &MsgArchiveCredentialSchema{}, "/vpr/v1/cs/archive-credential-schema")
}
