package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gftypes "github.com/verana-labs/verana/x/gf/types"
	trkeeper "github.com/verana-labs/verana/x/tr/keeper"
)

// TRAsEcosystemKeeper wraps the existing x/tr keeper to satisfy
// gftypes.EcosystemKeeper. Removed when issue #305 (TR→EC rename) lands —
// the renamed EC keeper will implement gftypes.EcosystemKeeper directly with
// the proper CorporationID FK.
//
// INTERIM: TR stores `corporation` as a bech32 address; MOD-GF needs
// corporation_id (uint64). Without MOD-CO (#303) to resolve address → id, we
// set CorporationID = 0 in the returned view. Any ecosystem-targeted GF
// subject check that compares `eco.CorporationID == co.id` will fail. This
// adapter is replaced wholesale when #305 lands; until then, EC-targeted
// MOD-GF calls are intentionally non-functional. Corporation-targeted MOD-GF
// calls are gated by StubCorporationKeeper below and are also non-functional.
type TRAsEcosystemKeeper struct {
	k trkeeper.Keeper
}

func NewTRAsEcosystemKeeper(k trkeeper.Keeper) gftypes.EcosystemKeeper {
	return TRAsEcosystemKeeper{k: k}
}

func (a TRAsEcosystemKeeper) GetEcosystemView(ctx context.Context, id uint64) (gftypes.EcosystemView, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	tr, err := a.k.GetTrustRegistry(sdkCtx, id)
	if err != nil {
		return gftypes.EcosystemView{}, false
	}
	return gftypes.EcosystemView{
		Id:            tr.Id,
		CorporationID: 0, // intentional — see file header
		Language:      tr.Language,
		ActiveVersion: tr.ActiveVersion,
	}, true
}

func (a TRAsEcosystemKeeper) SetEcosystemActiveVersion(ctx context.Context, id uint64, newVersion uint32) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	tr, err := a.k.GetTrustRegistry(sdkCtx, id)
	if err != nil {
		return fmt.Errorf("get trust registry %d: %w", id, err)
	}
	tr.ActiveVersion = newVersion
	tr.Modified = sdkCtx.BlockTime()
	return a.k.TrustRegistry.Set(sdkCtx, tr.Id, tr)
}

// StubCorporationKeeper returns (zero, false) for all lookups. Replaced when
// issue #303 (MOD-CO) lands; until then corporation-targeted MOD-GF calls
// abort cleanly with gftypes.ErrSubjectNotFound (because ResolveByPolicyAddress
// returns ok=false).
type StubCorporationKeeper struct{}

func NewStubCorporationKeeper() gftypes.CorporationKeeper {
	return StubCorporationKeeper{}
}

func (StubCorporationKeeper) ResolveByPolicyAddress(_ context.Context, _ string) (gftypes.CorporationView, bool) {
	return gftypes.CorporationView{}, false
}

func (StubCorporationKeeper) GetByID(_ context.Context, _ uint64) (gftypes.CorporationView, bool) {
	return gftypes.CorporationView{}, false
}

func (StubCorporationKeeper) SetActiveVersion(_ context.Context, _ uint64, _ uint32) error {
	return errors.New("corporation keeper not wired yet (MOD-CO pending in issue #303)")
}
