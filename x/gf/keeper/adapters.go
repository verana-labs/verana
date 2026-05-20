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
// the renamed EC keeper will implement gftypes.EcosystemKeeper directly.
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
		Corporation:   tr.Corporation,
		Language:      tr.Language,
		ActiveVersion: tr.ActiveVersion,
	}, true
}

func (a TRAsEcosystemKeeper) SetEcosystemActiveVersion(ctx context.Context, id uint64, newVersion int32) error {
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
// abort cleanly with gftypes.ErrSubjectNotFound (because GetCorporationView
// returns ok=false).
type StubCorporationKeeper struct{}

func NewStubCorporationKeeper() gftypes.CorporationKeeper {
	return StubCorporationKeeper{}
}

func (StubCorporationKeeper) GetCorporationView(ctx context.Context, _ string) (gftypes.CorporationView, bool) {
	return gftypes.CorporationView{}, false
}

func (StubCorporationKeeper) SetCorporationActiveVersion(ctx context.Context, _ string, _ int32) error {
	return errors.New("corporation keeper not wired yet (MOD-CO pending in issue #303)")
}
