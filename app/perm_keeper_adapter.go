package app

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	permkeeper "github.com/verana-labs/verana/x/perm/keeper"
)

// permKeeperAdapter wraps the perm module keeper to satisfy the DE module's
// PermKeeper interface without making permkeeper.Keeper directly implement it.
// This avoids a depinject cyclic dependency (TR → DE → Perm → TR) that would
// occur if permkeeper.Keeper satisfied de.types.PermKeeper via duck typing.
type permKeeperAdapter struct {
	pk permkeeper.Keeper
}

func (a *permKeeperAdapter) GetPermissionForVSOA(ctx context.Context, permID uint64) (authority string, vsOperator string, withFeegrant bool, effectiveUntil *time.Time, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	perm, err := a.pk.GetPermissionByID(sdkCtx, permID)
	if err != nil {
		return "", "", false, nil, err
	}
	return perm.Authority, perm.VsOperator, perm.VsOperatorAuthzWithFeegrant, perm.EffectiveUntil, nil
}
