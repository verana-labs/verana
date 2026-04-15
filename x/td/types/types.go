package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic implements sdk.Msg
func (msg *MsgReclaimTrustDepositYield) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid corporation address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
	}
	if msg.Amount == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "amount must be > 0")
	}
	return nil
}

func (msg *MsgSlashTrustDeposit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid corporation address (%s)", err)
	}

	if msg.Deposit.IsZero() || msg.Deposit.IsNegative() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "deposit must be greater than 0")
	}

	return nil
}

func (msg *MsgRepaySlashedTrustDeposit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Corporation)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid corporation address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid operator address (%s)", err)
	}

	if msg.Deposit == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "deposit must be greater than 0")
	}

	return nil
}
