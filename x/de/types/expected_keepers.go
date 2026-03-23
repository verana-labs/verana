package types

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuthKeeper defines the expected interface for the Auth module.
type AuthKeeper interface {
	AddressCodec() address.Codec
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

// PermKeeper defines the expected interface for the Permission module.
// Used by [MOD-DE-MSG-5] and [MOD-DE-MSG-6] to load permission data for
// VS operator authorization management.
type PermKeeper interface {
	// GetPermissionForVSOA returns the fields needed for VS operator authorization.
	// Returns authority, vsOperator, vsOperatorAuthzWithFeegrant, effectiveUntil, err.
	GetPermissionForVSOA(ctx context.Context, permID uint64) (authority string, vsOperator string, withFeegrant bool, effectiveUntil *time.Time, err error)
}
