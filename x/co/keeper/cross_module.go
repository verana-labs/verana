package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// CoAsGFCorporationKeeper adapts the MOD-CO keeper to MOD-GF's CorporationKeeper
// interface. Wired post-construction via gfKeeper.SetCorporationKeeper to break
// the MOD-GF ↔ MOD-CO depinject cycle.
type CoAsGFCorporationKeeper struct {
	k Keeper
}

func NewCoAsGFCorporationKeeper(k Keeper) gftypes.CorporationKeeper {
	return CoAsGFCorporationKeeper{k: k}
}

// ResolveByPolicyAddress backs MOD-GF AUTHZ-CHECK-5.
func (a CoAsGFCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (gftypes.CorporationView, bool) {
	coID, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return gftypes.CorporationView{}, false
	}
	co, err := a.k.Corporation.Get(ctx, coID)
	if err != nil {
		return gftypes.CorporationView{}, false
	}
	return gftypes.CorporationView{
		Id:            co.Id,
		PolicyAddress: co.PolicyAddress,
		Language:      co.Language,
		ActiveVersion: co.ActiveVersion,
	}, true
}

func (a CoAsGFCorporationKeeper) GetByID(ctx context.Context, corporationID uint64) (gftypes.CorporationView, bool) {
	co, err := a.k.Corporation.Get(ctx, corporationID)
	if err != nil {
		return gftypes.CorporationView{}, false
	}
	return gftypes.CorporationView{
		Id:            co.Id,
		PolicyAddress: co.PolicyAddress,
		Language:      co.Language,
		ActiveVersion: co.ActiveVersion,
	}, true
}

// SetActiveVersion is called by MOD-GF MSG-2 (IncreaseActiveGovernanceFrameworkVersion).
func (a CoAsGFCorporationKeeper) SetActiveVersion(ctx context.Context, corporationID uint64, newVersion uint32) error {
	co, err := a.k.Corporation.Get(ctx, corporationID)
	if err != nil {
		return fmt.Errorf("corporation %d not found: %w", corporationID, err)
	}
	co.ActiveVersion = newVersion
	co.Modified = sdk.UnwrapSDKContext(ctx).BlockTime()
	return a.k.Corporation.Set(ctx, co.Id, co)
}
