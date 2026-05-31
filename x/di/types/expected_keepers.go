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

// DelegationKeeper defines the expected interface for the Delegation (DE) module.
// Used to perform [AUTHZ-CHECK] for (authority, operator) pairs.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, authority string, operator string, msgTypeURL string, now time.Time) error
}

// CorporationView is the read shape MOD-DI needs about a Corporation subject
// for AUTHZ-CHECK-5: the signing `authority` policy_address resolved to its id.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
}

// CorporationKeeper backs AUTHZ-CHECK-5 for MOD-DI's delegable MsgStoreDigest:
// resolve the signing `authority` policy_address to its registered Corporation,
// or abort with ErrCorporationNotRegistered (referencing MOD-CO-MSG-1).
type CorporationKeeper interface {
	ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
