package types

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// AccountKeeper defines the expected interface for the Account module
// (sim-only — keeper code doesn't depend on it).
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected interface for the Bank module
// (sim-only — keeper code doesn't depend on it).
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
}

// DelegationKeeper backs AUTHZ-CHECK for delegable Msgs in x/ec.
// Method signature matches x/de keeper exactly so depinject can auto-wire it.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, corporation string, operator string, msgTypeURL string, now time.Time) error
}

// CorporationView is the read shape MOD-ES needs about a Corporation subject.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
	Language      string
	ActiveVersion uint32
}

// CorporationKeeper backs AUTHZ-CHECK-5 for MOD-ES messages: resolve the
// signing `corporation` policy_address to the Corporation entry, and produce
// its uint64 id for `ecosystem.corporation_id` writes / ownership checks.
type CorporationKeeper interface {
	ResolveByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, bool)
	GetByID(ctx context.Context, corporationID uint64) (CorporationView, bool)
}

// GFKeeper is the cross-module surface MOD-ES needs from MOD-GF:
//   - CreateInitialGFVersionForEcosystem seeds v1 GFV+GFD for a new Ecosystem
//     (called from MSG-1 execution).
//   - ListVersionsByEcosystem returns nested GFV+GFD for QRY-1 / QRY-2.
//
// SetEcosystemActiveVersion is the OPPOSITE direction — MOD-GF calls it via
// the gftypes.EcosystemKeeper interface that x/ec provides; it's not on this
// interface.
type GFKeeper interface {
	CreateInitialGFVersionForEcosystem(ctx context.Context, ecosystemID uint64, language, docURL, docDigestSRI string) error
	ListVersionsByEcosystem(ctx context.Context, ecosystemID uint64, activeVersion uint32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error)
}
