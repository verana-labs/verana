package keeper

import "github.com/verana-labs/verana/x/co/types"

type querier struct {
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &querier{Keeper: k}
}

var _ types.QueryServer = querier{}
