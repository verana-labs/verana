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
		&MsgCreateTrustRegistry{},
		&MsgAddGovernanceFrameworkDocument{},
		&MsgIncreaseActiveGovernanceFrameworkVersion{},
		&MsgUpdateTrustRegistry{},
		&MsgArchiveTrustRegistry{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgCreateTrustRegistry{}, "/vpr/v1/tr/create-trust-registry")
	legacy.RegisterAminoMsg(cdc, &MsgAddGovernanceFrameworkDocument{}, "/vpr/v1/tr/add-gfd")
	legacy.RegisterAminoMsg(cdc, &MsgIncreaseActiveGovernanceFrameworkVersion{}, "/vpr/v1/tr/increase-active-gf-version")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateTrustRegistry{}, "/vpr/v1/tr/update-trust-registry")

}
