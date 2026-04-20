package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// [MOD-XR-*] Register messages in the legacy amino codec so Ledger / amino-based
// governance flows can sign them.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "verana/x/xr/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgCreateExchangeRate{}, "verana/x/xr/MsgCreateExchangeRate")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateExchangeRate{}, "verana/x/xr/MsgUpdateExchangeRate")
	legacy.RegisterAminoMsg(cdc, &MsgSetExchangeRateState{}, "verana/x/xr/MsgSetExchangeRateState")
}

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateExchangeRate{},
		&MsgUpdateExchangeRate{},
		&MsgSetExchangeRateState{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
