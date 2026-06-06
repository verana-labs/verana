package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	detypes "github.com/verana-labs/verana/x/de/types"
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

// CoAsDeCorporationKeeper adapts the MOD-CO keeper to MOD-DE's CorporationKeeper
// interface. Wired post-construction via deKeeper.SetCorporationKeeper to break
// the MOD-DE ↔ MOD-CO depinject cycle (MOD-CO depends on MOD-DE's
// DelegationKeeper for AUTHZ-CHECK-1).
type CoAsDeCorporationKeeper struct {
	k Keeper
}

func NewCoAsDeCorporationKeeper(k Keeper) detypes.CorporationKeeper {
	return CoAsDeCorporationKeeper{k: k}
}

// ResolveCorporationByPolicyAddress backs MOD-DE AUTHZ-CHECK-5, routing through
// the canonical resolver so an unregistered signer aborts with
// ErrCorporationNotRegistered (referencing MOD-CO-MSG-1).
func (a CoAsDeCorporationKeeper) ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (detypes.CorporationView, error) {
	co, err := a.k.ResolveCorporationByPolicyAddress(ctx, policyAddress)
	if err != nil {
		return detypes.CorporationView{}, err
	}
	return detypes.CorporationView{Id: co.Id, PolicyAddress: co.PolicyAddress}, nil
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
