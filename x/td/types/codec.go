package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	// this line is used by starport scaffolding # 1
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgReclaimTrustDepositYield{}, "/td/v1/reclaim-interests")
	legacy.RegisterAminoMsg(cdc, &MsgReclaimTrustDeposit{}, "/td/v1/reclaim-deposit")
	legacy.RegisterAminoMsg(cdc, &MsgRepaySlashedTrustDeposit{}, "/td/v1/repay-slashed-td")
	legacy.RegisterAminoMsg(cdc, &SlashTrustDepositProposal{}, "td/v1/SlashTrustDepositProposal")

}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&SlashTrustDepositProposal{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgReclaimTrustDepositYield{},
		&MsgReclaimTrustDeposit{},
		&MsgRepaySlashedTrustDeposit{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
