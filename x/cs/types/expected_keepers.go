package types

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ectypes "github.com/verana-labs/verana/x/ec/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// EcosystemKeeper is the cross-module surface MOD-CS needs from MOD-ES:
// fetch the Ecosystem row that owns a CredentialSchema (by ecosystem_id) so
// the ownership check can compare ec.CorporationId against the resolved
// signing corporation. Replaces the legacy TrustRegistryKeeper.
type EcosystemKeeper interface {
	Ecosystem
}

// Ecosystem is the read shape MOD-CS needs from MOD-ES. Method names are
// expressed on EcosystemKeeper above; this nested-interface form keeps the
// imported types out of the public method signature in autodiscovery tools.
type Ecosystem interface {
	GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error)
}

// CorporationView is the read shape MOD-CS needs about a Corporation subject.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
}

// CorporationKeeper backs the AUTHZ-CHECK-5 resolution for MOD-CS messages:
// turn the signing `corporation` policy_address into the uint64 co.id used to
// validate ec.CorporationId ownership.
type CorporationKeeper interface {
	ResolveByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, bool)
}

// DelegationKeeper backs AUTHZ-CHECK for delegable Msgs in x/cs.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, authority string, operator string, msgTypeURL string, now time.Time) error
}
