package types

import (
	"context"

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

// CorporationView is the read shape MOD-DE needs about a Corporation subject
// for AUTHZ-CHECK-5: the signing `corporation` policy_address resolved to its id.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
}

// CorporationKeeper backs AUTHZ-CHECK-5 for MOD-DE's delegable Grant/Revoke
// messages: resolve the signing `corporation` policy_address to its registered
// Corporation, or abort with ErrCorporationNotRegistered (referencing
// MOD-CO-MSG-1). Wired post-construction via Keeper.SetCorporationKeeper to
// break the MOD-DE ↔ MOD-CO depinject cycle.
type CorporationKeeper interface {
	ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
}
