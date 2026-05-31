package keeper

import (
	"context"

	cokeeper "github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/di/types"
)

// CoAsDICorporationKeeper adapts the concrete x/co keeper to
// ditypes.CorporationKeeper. It routes AUTHZ-CHECK-5 through the canonical
// cokeeper.ResolveCorporationByPolicyAddress, so an unregistered signer aborts
// with ErrCorporationNotRegistered (referencing MOD-CO-MSG-1).
type CoAsDICorporationKeeper struct {
	k cokeeper.Keeper
}

func NewCoAsDICorporationKeeper(k cokeeper.Keeper) types.CorporationKeeper {
	return CoAsDICorporationKeeper{k: k}
}

func (a CoAsDICorporationKeeper) ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, error) {
	co, err := a.k.ResolveCorporationByPolicyAddress(ctx, policyAddress)
	if err != nil {
		return types.CorporationView{}, err
	}
	return types.CorporationView{Id: co.Id, PolicyAddress: co.PolicyAddress}, nil
}
