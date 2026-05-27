package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "verana/x/ec/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgCreateEcosystem{}, "verana/x/ec/MsgCreateEcosystem")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateEcosystem{}, "verana/x/ec/MsgUpdateEcosystem")
	legacy.RegisterAminoMsg(cdc, &MsgArchiveEcosystem{}, "verana/x/ec/MsgArchiveEcosystem")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateEcosystem{},
		&MsgUpdateEcosystem{},
		&MsgArchiveEcosystem{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
