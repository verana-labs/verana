package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cokeeper "github.com/verana-labs/verana/x/co/keeper"
	eckeeper "github.com/verana-labs/verana/x/ec/keeper"
	ectypes "github.com/verana-labs/verana/x/ec/types"
	"github.com/verana-labs/verana/x/perm/types"
)

// EcAsPermEcosystemKeeper adapts x/ec keeper to permtypes.EcosystemKeeper,
// supplying both GetEcosystem (for ownership checks) and GetTrustUnitPrice
// (for fee math).
type EcAsPermEcosystemKeeper struct {
	k eckeeper.Keeper
}

func NewEcAsPermEcosystemKeeper(k eckeeper.Keeper) types.EcosystemKeeper {
	return EcAsPermEcosystemKeeper{k: k}
}

func (a EcAsPermEcosystemKeeper) GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error) {
	return a.k.GetEcosystem(ctx, id)
}

func (a EcAsPermEcosystemKeeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return a.k.GetTrustUnitPrice(ctx)
}

// CoAsPermCorporationKeeper adapts x/co keeper to permtypes.CorporationKeeper.
type CoAsPermCorporationKeeper struct {
	k cokeeper.Keeper
}

func NewCoAsPermCorporationKeeper(k cokeeper.Keeper) types.CorporationKeeper {
	return CoAsPermCorporationKeeper{k: k}
}

func (a CoAsPermCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	coID, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{Id: coID, PolicyAddress: policyAddress}, true
}
