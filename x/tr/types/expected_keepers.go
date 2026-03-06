package types

import (
	"context"
	"time"

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

// DelegationKeeper defines the expected interface for the Delegation (DE) module.
// Used to perform [AUTHZ-CHECK] for (authority, operator) pairs.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, authority string, operator string, msgTypeURL string, now time.Time) error
}
