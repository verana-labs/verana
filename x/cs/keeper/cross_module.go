package keeper

import (
	"context"

	cokeeper "github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/cs/types"
	eckeeper "github.com/verana-labs/verana/x/ec/keeper"
	ectypes "github.com/verana-labs/verana/x/ec/types"
)

// EcAsCSEcosystemKeeper adapts x/ec keeper to cstypes.EcosystemKeeper. The
// MOD-CS Ecosystem interface only needs GetEcosystem; ec keeper has it.
type EcAsCSEcosystemKeeper struct {
	k eckeeper.Keeper
}

func NewEcAsCSEcosystemKeeper(k eckeeper.Keeper) types.EcosystemKeeper {
	return EcAsCSEcosystemKeeper{k: k}
}

func (a EcAsCSEcosystemKeeper) GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error) {
	return a.k.GetEcosystem(ctx, id)
}

// CoAsCSCorporationKeeper adapts x/co keeper to cstypes.CorporationKeeper.
// MOD-CS only needs Id + PolicyAddress for its ownership-chain check.
type CoAsCSCorporationKeeper struct {
	k cokeeper.Keeper
}

func NewCoAsCSCorporationKeeper(k cokeeper.Keeper) types.CorporationKeeper {
	return CoAsCSCorporationKeeper{k: k}
}

func (a CoAsCSCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	coID, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{Id: coID, PolicyAddress: policyAddress}, true
}
