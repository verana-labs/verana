package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"

	"github.com/verana-labs/verana/x/co/types"
	gfkeeper "github.com/verana-labs/verana/x/gf/keeper"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// GroupKeeperAdapter narrows the SDK's x/group keeper down to the
// types.GroupKeeper surface that MOD-CO actually uses.
type GroupKeeperAdapter struct {
	k groupkeeper.Keeper
}

func NewGroupKeeperAdapter(k groupkeeper.Keeper) types.GroupKeeper {
	return GroupKeeperAdapter{k: k}
}

func (a GroupKeeperAdapter) CreateGroupWithPolicy(ctx context.Context, req *group.MsgCreateGroupWithPolicy) (*group.MsgCreateGroupWithPolicyResponse, error) {
	return a.k.CreateGroupWithPolicy(ctx, req)
}

// GFKeeperAdapter narrows the MOD-GF keeper down to the types.GFKeeper surface
// that MOD-CO actually uses (initial-version seed + per-corporation listing).
type GFKeeperAdapter struct {
	k gfkeeper.Keeper
}

func NewGFKeeperAdapter(k gfkeeper.Keeper) types.GFKeeper {
	return GFKeeperAdapter{k: k}
}

func (a GFKeeperAdapter) CreateInitialGFVersionForCorporation(ctx context.Context, corpID uint64, language, docURL, docDigestSRI string) error {
	return a.k.CreateInitialGFVersionForCorporation(ctx, corpID, language, docURL, docDigestSRI)
}

func (a GFKeeperAdapter) ListVersionsByCorporation(ctx context.Context, corpID uint64, activeVersion uint32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	return a.k.ListVersionsByCorporation(ctx, corpID, activeVersion, activeOnly, preferredLang)
}
