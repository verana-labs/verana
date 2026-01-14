package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// TrustDepositKeeper defines the expected interface for the Trust Deposit module.
type TrustDepositKeeper interface {
	AdjustTrustDeposit(ctx sdk.Context, account string, augend int64) error
	// Anchor-aware methods for Issue #185
	AdjustAnchorTrustDeposit(ctx sdk.Context, anchorID string, augend int64, operatorAccount string) error
	IsAnchor(ctx context.Context, anchorID string) bool
	GetAnchorForOperator(ctx context.Context, operatorAccount string) (string, error)
}
