package types

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/x/group"

	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// DelegationKeeper is the minimum surface MOD-CO needs from x/de for
// AUTHZ-CHECK-1 (operator authorization on MSG-2). Signature matches
// x/de/keeper.Keeper.CheckOperatorAuthorization exactly so the DE keeper
// concrete type satisfies this interface and depinject can auto-wire it.
type DelegationKeeper interface {
	CheckOperatorAuthorization(ctx context.Context, corporation string, operator string, msgTypeURL string, now time.Time) error
}

// GroupKeeper is the minimum surface MOD-CO needs from x/group for MSG-1.
// `CreateGroupWithPolicy` is a public method on x/group/keeper.Keeper that
// atomically creates a Group, a GroupPolicy, and (with group_policy_as_admin
// set) makes the policy address the admin of both — exactly the shape MOD-CO
// needs for the "group + Corporation in one tx" flow.
type GroupKeeper interface {
	CreateGroupWithPolicy(ctx context.Context, req *group.MsgCreateGroupWithPolicy) (*group.MsgCreateGroupWithPolicyResponse, error)
}

// GFKeeper is the minimum surface MOD-CO needs from x/gf for:
//   - MSG-1 post-commit step: seed the first GovernanceFrameworkVersion +
//     GovernanceFrameworkDocument bound to the new Corporation.
//   - Query enrichment: list the Corporation's GF versions + docs for the
//     CorporationWithGF response shape.
//
// The MOD-CO ↔ MOD-GF construction-time cycle is broken by giving MOD-GF a
// SetCorporationKeeper setter that runs after both keepers are built; this
// interface is the read direction (MOD-CO calling MOD-GF) and has no cycle
// concern.
type GFKeeper interface {
	CreateInitialGFVersionForCorporation(ctx context.Context, corpID uint64, language, docURL, docDigestSRI string) error
	ListVersionsByCorporation(ctx context.Context, corpID uint64, activeVersion uint32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error)
}
