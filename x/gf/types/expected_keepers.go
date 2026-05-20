package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DelegationKeeper is the minimum surface MOD-GF needs from x/de for AUTHZ-CHECK-1.
// Mirrors x/tr/types.DelegationKeeper.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx sdk.Context, corporation string, operator string, msgType string) error
}

// EcosystemView is the read shape MOD-GF needs to validate ecosystem subjects.
// `Corporation` is the controlling corporation (group_policy_address).
// `Language` is the ecosystem's primary language.
// `ActiveVersion` is the ecosystem's current active GF version.
type EcosystemView struct {
	Id            uint64
	Corporation   string
	Language      string
	ActiveVersion int32
}

// EcosystemKeeper is the minimum surface MOD-GF needs for ecosystem-targeted GF ops.
// Until issue #305 (TR→EC rename) lands, the x/tr keeper provides this via an adapter.
type EcosystemKeeper interface {
	GetEcosystemView(ctx context.Context, ecosystemID uint64) (EcosystemView, bool)
	SetEcosystemActiveVersion(ctx context.Context, ecosystemID uint64, newVersion int32) error
}

// CorporationView is the read shape MOD-GF needs to validate corporation subjects.
// ActiveVersion uses int32 to match EcosystemView and the underlying spec
// (`active_version (int)`), preventing silent overflow on cast in resolveSubject.
type CorporationView struct {
	GroupPolicyAddress string
	Language           string
	ActiveVersion      int32
}

// CorporationKeeper is the minimum surface MOD-GF needs for corporation-targeted GF ops.
// Until issue #303 (MOD-CO) lands, a stub keeper returns (zero, false) for all calls.
// The implementation MUST also bump `corp.modified` when active_version is updated
// (per MOD-GF-MSG-2-3 step "Set subject.modified to current timestamp").
type CorporationKeeper interface {
	GetCorporationView(ctx context.Context, groupPolicyAddress string) (CorporationView, bool)
	SetCorporationActiveVersion(ctx context.Context, groupPolicyAddress string, newVersion int32) error
}
