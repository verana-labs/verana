package ecosystem

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
)

// InitGenesis loads Ecosystem entries + the (did, corp_id) consistency index
// + counters. GFV/GFD belong to x/gf and are loaded by that module.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("set params: %s", err))
	}
	for _, ec := range gs.Ecosystems {
		if err := k.Ecosystem.Set(ctx, ec.Id, ec); err != nil {
			panic(fmt.Sprintf("set ecosystem %d: %s", ec.Id, err))
		}
		if err := k.EcosystemByDIDCorp.Set(ctx, collections.Join(ec.Did, ec.CorporationId), ec.Id); err != nil {
			panic(fmt.Sprintf("set (did,corp) index for ecosystem %d: %s", ec.Id, err))
		}
	}
	for _, c := range gs.Counters {
		if err := k.Counter.Set(ctx, c.EntityType, c.Value); err != nil {
			panic(fmt.Sprintf("set counter %s: %s", c.EntityType, err))
		}
	}
}

// ExportGenesis exports the Ecosystem entries and the "ec" counter only.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.Params = k.GetParams(ctx)

	var ecosystems []types.Ecosystem
	if err := k.Ecosystem.Walk(ctx, nil, func(_ uint64, ec types.Ecosystem) (bool, error) {
		ecosystems = append(ecosystems, ec)
		return false, nil
	}); err != nil {
		panic(fmt.Sprintf("walk ecosystems: %s", err))
	}
	gs.Ecosystems = ecosystems

	if v, err := k.Counter.Get(ctx, "ec"); err == nil {
		gs.Counters = append(gs.Counters, types.Counter{EntityType: "ec", Value: v})
	}

	return gs
}
