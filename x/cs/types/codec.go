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
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "verana/x/cs/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgCreateCredentialSchema{}, "verana/x/cs/MsgCreateCredentialSchema")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateCredentialSchema{}, "verana/x/cs/MsgUpdateCredentialSchema")
	legacy.RegisterAminoMsg(cdc, &MsgArchiveCredentialSchema{}, "verana/x/cs/MsgArchiveCredentialSchema")
	legacy.RegisterAminoMsg(cdc, &MsgCreateSchemaAuthorizationPolicy{}, "verana/x/cs/MsgCreateSchemaAuthPolicy")
	legacy.RegisterAminoMsg(cdc, &MsgIncreaseActiveSchemaAuthorizationPolicyVersion{}, "verana/x/cs/MsgIncSchemaAuthPolicyVer")
	legacy.RegisterAminoMsg(cdc, &MsgRevokeSchemaAuthorizationPolicy{}, "verana/x/cs/MsgRevokeSchemaAuthPolicy")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateCredentialSchema{},
		&MsgUpdateCredentialSchema{},
		&MsgArchiveCredentialSchema{},
		&MsgCreateSchemaAuthorizationPolicy{},
		&MsgIncreaseActiveSchemaAuthorizationPolicyVersion{},
		&MsgRevokeSchemaAuthorizationPolicy{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
