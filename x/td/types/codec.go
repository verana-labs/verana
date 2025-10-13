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
	legacy.RegisterAminoMsg(cdc, &MsgReclaimTrustDepositYield{}, "/td/v1/reclaim-interests")
	legacy.RegisterAminoMsg(cdc, &MsgReclaimTrustDeposit{}, "/td/v1/reclaim-deposit")
	legacy.RegisterAminoMsg(cdc, &MsgSlashTrustDeposit{}, "/td/v1/slash-td")
	legacy.RegisterAminoMsg(cdc, &MsgRepaySlashedTrustDeposit{}, "/td/v1/repay-slashed-td")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgReclaimTrustDepositYield{},
		&MsgReclaimTrustDeposit{},
		&MsgSlashTrustDeposit{},
		&MsgRepaySlashedTrustDeposit{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
