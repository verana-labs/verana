package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic implements sdk.Msg
func (msg *MsgReclaimTrustDepositYield) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return fmt.Errorf("invalid creator address (%s)", err)
	}
	return nil
}

func (msg *MsgReclaimTrustDeposit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return fmt.Errorf("invalid creator address (%s)", err)
	}
	if msg.Claimed == 0 {
		return fmt.Errorf("claimed amount must be greater than 0")
	}
	return nil
}

func (msg *MsgRepaySlashedTrustDeposit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Account)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid account address (%s)", err)
	}

	if msg.Amount == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "amount must be greater than 0")
	}

	return nil
}
