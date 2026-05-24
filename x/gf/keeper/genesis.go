package keeper

import (
	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) error {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		return err
	}
	var maxGFV, maxGFD uint64
	for _, gfv := range gs.Versions {
		if err := k.GFVersion.Set(ctx, gfv.Id, gfv); err != nil {
			return err
		}
		if gfv.EcosystemId > 0 {
			if err := k.GFVersionByEcosystem.Set(ctx, collections.Join(gfv.EcosystemId, gfv.Version), gfv.Id); err != nil {
				return err
			}
		} else {
			if err := k.GFVersionByCorporation.Set(ctx, collections.Join(gfv.CorporationId, gfv.Version), gfv.Id); err != nil {
				return err
			}
		}
		if gfv.Id > maxGFV {
			maxGFV = gfv.Id
		}
	}
	for _, gfd := range gs.Documents {
		if err := k.GFDocument.Set(ctx, gfd.Id, gfd); err != nil {
			return err
		}
		if gfd.Id > maxGFD {
			maxGFD = gfd.Id
		}
	}
	if err := k.Counter.Set(ctx, "gfv", maxGFV); err != nil {
		return err
	}
	if err := k.Counter.Set(ctx, "gfd", maxGFD); err != nil {
		return err
	}
	return nil
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	gs := &types.GenesisState{
		Params: k.GetParams(ctx),
	}
	_ = k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		gs.Versions = append(gs.Versions, gfv)
		return false, nil
	})
	_ = k.GFDocument.Walk(ctx, nil, func(_ uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
		gs.Documents = append(gs.Documents, gfd)
		return false, nil
	})
	return gs
}
