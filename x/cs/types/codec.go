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
	legacy.RegisterAminoMsg(cdc, &MsgCreateCredentialSchema{}, "/vpr/v1/cs/create-credential-schema")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateCredentialSchema{}, "/vpr/v1/cs/update-credential-schema")
	legacy.RegisterAminoMsg(cdc, &MsgArchiveCredentialSchema{}, "/vpr/v1/cs/archive-credential-schema")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateCredentialSchema{},
		&MsgUpdateCredentialSchema{},
		&MsgArchiveCredentialSchema{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
