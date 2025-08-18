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
		&MsgReclaimTrustDepositYield{},
		&MsgReclaimTrustDeposit{},
		&MsgRepaySlashedTrustDeposit{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgReclaimTrustDepositYield{}, "/td/v1/reclaim-interests")
	legacy.RegisterAminoMsg(cdc, &MsgReclaimTrustDeposit{}, "/td/v1/reclaim-deposit")
	legacy.RegisterAminoMsg(cdc, &MsgRepaySlashedTrustDeposit{}, "/td/v1/repay-slashed-td")
	legacy.RegisterAminoMsg(cdc, &SlashTrustDepositProposal{}, "td/v1/SlashTrustDepositProposal")

}
