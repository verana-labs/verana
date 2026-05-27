package types

import (
	"context"
	"time"
)

// DelegationKeeper is the minimum surface MOD-GF needs from x/de for AUTHZ-CHECK-1.
// Signature matches x/de/keeper.Keeper.CheckOperatorAuthorization exactly so the
// DE keeper concrete type satisfies this interface and depinject can auto-wire it.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, corporation string, operator string, msgTypeURL string, now time.Time) error
}

// EcosystemView is the read shape MOD-GF needs to validate ecosystem subjects.
// `CorporationID` is the uint64 FK to the controlling Corporation.
// `Language` is the ecosystem's primary language.
// `ActiveVersion` is the ecosystem's current active GF version.
type EcosystemView struct {
	Id            uint64
	CorporationID uint64
	Language      string
	ActiveVersion uint32
}

// EcosystemKeeper is the minimum surface MOD-GF needs for ecosystem-targeted
// GF ops. Provided by x/ec via EcAsGFEcosystemKeeper after the TR→EC rename
// (issue #305). `CorporationID` on the returned view is the real FK to the
// controlling Corporation; subject-controller checks compare against it.
type EcosystemKeeper interface {
	GetEcosystemView(ctx context.Context, ecosystemID uint64) (EcosystemView, bool)
	SetEcosystemActiveVersion(ctx context.Context, ecosystemID uint64, newVersion uint32) error
}

// CorporationView is the read shape MOD-GF needs about a Corporation subject.
// `Id` is the canonical uint64 FK target everywhere in VPR.
// `PolicyAddress` is the on-chain account that signs on the Corporation's behalf.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
	Language      string
	ActiveVersion uint32
}

// CorporationKeeper is the minimum surface MOD-GF needs for:
//   - AUTHZ-CHECK-5 resolution: ResolveByPolicyAddress takes the signing account
//     (= policy_address) and returns the registered Corporation (its id, language,
//     active_version). MUST be called at the entry of every delegable MOD-GF Msg.
//   - Subject lookups by id: GetByID returns a Corporation by its uint64 id (used
//     by the query layer for the active_only filter).
//   - Active version mutation: SetActiveVersion bumps `co.active_version` and
//     `co.modified` per MOD-GF-MSG-2-3.
//
// Until issue #303 (MOD-CO) lands, a stub keeper returns (zero, false) for all
// lookups so all corporation-targeted MOD-GF calls abort with ErrSubjectNotFound.
type CorporationKeeper interface {
	ResolveByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, bool)
	GetByID(ctx context.Context, corporationID uint64) (CorporationView, bool)
	SetActiveVersion(ctx context.Context, corporationID uint64, newVersion uint32) error
}
