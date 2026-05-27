package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cokeeper "github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/ec/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

// EcAsGFEcosystemKeeper adapts the MOD-ES keeper to gftypes.EcosystemKeeper,
// supplying MOD-GF with the surface it needs for ecosystem-targeted GF
// operations: GetEcosystemView (read shape including CorporationID for
// subject-controller checks) and SetEcosystemActiveVersion (called from
// MOD-GF MSG-2 to bump ec.active_version after IncreaseActiveGFVersion).
//
// Replaces the interim TRAsEcosystemKeeper which returned CorporationID=0;
// EC natively has the uint64 FK so the controller check now functions.
type EcAsGFEcosystemKeeper struct {
	k Keeper
}

func NewEcAsGFEcosystemKeeper(k Keeper) gftypes.EcosystemKeeper { return EcAsGFEcosystemKeeper{k: k} }

func (a EcAsGFEcosystemKeeper) GetEcosystemView(ctx context.Context, ecosystemID uint64) (gftypes.EcosystemView, bool) {
	ec, err := a.k.Ecosystem.Get(ctx, ecosystemID)
	if err != nil {
		return gftypes.EcosystemView{}, false
	}
	return gftypes.EcosystemView{
		Id:            ec.Id,
		CorporationID: ec.CorporationId,
		Language:      ec.Language,
		ActiveVersion: ec.ActiveVersion,
	}, true
}

func (a EcAsGFEcosystemKeeper) SetEcosystemActiveVersion(ctx context.Context, ecosystemID uint64, newVersion uint32) error {
	ec, err := a.k.Ecosystem.Get(ctx, ecosystemID)
	if err != nil {
		if cerrIsNotFound(err) {
			return fmt.Errorf("ecosystem %d not found: %w", ecosystemID, err)
		}
		return fmt.Errorf("get ecosystem %d: %w", ecosystemID, err)
	}
	ec.ActiveVersion = newVersion
	ec.Modified = sdk.UnwrapSDKContext(ctx).BlockTime()
	return a.k.Ecosystem.Set(ctx, ec.Id, ec)
}

func cerrIsNotFound(err error) bool {
	return err != nil && (err == collections.ErrNotFound || err.Error() == collections.ErrNotFound.Error())
}

// CoAsECCorporationKeeper adapts x/co keeper to ectypes.CorporationKeeper
// for MOD-ES AUTHZ-CHECK-5. The ectypes.CorporationView shape matches the
// gftypes.CorporationView shape; this adapter exists purely to bridge the
// two distinct types.
type CoAsECCorporationKeeper struct {
	k cokeeper.Keeper
}

func NewCoAsECCorporationKeeper(k cokeeper.Keeper) types.CorporationKeeper {
	return CoAsECCorporationKeeper{k: k}
}

func (a CoAsECCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	coID, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return types.CorporationView{}, false
	}
	co, err := a.k.Corporation.Get(ctx, coID)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{
		Id:            co.Id,
		PolicyAddress: co.PolicyAddress,
		Language:      co.Language,
		ActiveVersion: co.ActiveVersion,
	}, true
}

func (a CoAsECCorporationKeeper) GetByID(ctx context.Context, corporationID uint64) (types.CorporationView, bool) {
	co, err := a.k.Corporation.Get(ctx, corporationID)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{
		Id:            co.Id,
		PolicyAddress: co.PolicyAddress,
		Language:      co.Language,
		ActiveVersion: co.ActiveVersion,
	}, true
}
