package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/co/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}
	var maxID uint64
	for _, co := range gs.Corporations {
		if err := k.Corporation.Set(ctx, co.Id, co); err != nil {
			return err
		}
		if err := k.CorporationByPolicyAddr.Set(ctx, co.PolicyAddress, co.Id); err != nil {
			return err
		}
		if err := k.CorporationByDID.Set(ctx, co.Did, co.Id); err != nil {
			return err
		}
		if co.Id > maxID {
			maxID = co.Id
		}
	}
	return k.Counter.Set(ctx, "co", maxID)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{Params: k.GetParams(ctx)}
	_ = k.Corporation.Walk(ctx, nil, func(_ uint64, co types.Corporation) (bool, error) {
		gs.Corporations = append(gs.Corporations, co)
		return false, nil
	})
	return gs
}
