package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/gf/types"
)

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	p, err := k.Params.Get(ctx)
	if err != nil {
		return types.DefaultParams()
	}
	return p
}

func (k Keeper) SetParams(ctx sdk.Context, p types.Params) error {
	return k.Params.Set(ctx, p)
}
