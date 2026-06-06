package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cokeeper "github.com/verana-labs/verana/x/co/keeper"
	eckeeper "github.com/verana-labs/verana/x/ec/keeper"
	ectypes "github.com/verana-labs/verana/x/ec/types"
	"github.com/verana-labs/verana/x/pp/types"
)

// EcAsParticipantEcosystemKeeper adapts x/ec keeper to participanttypes.EcosystemKeeper,
// supplying both GetEcosystem (for ownership checks) and GetTrustUnitPrice
// (for fee math).
type EcAsParticipantEcosystemKeeper struct {
	k eckeeper.Keeper
}

func NewEcAsParticipantEcosystemKeeper(k eckeeper.Keeper) types.EcosystemKeeper {
	return EcAsParticipantEcosystemKeeper{k: k}
}

func (a EcAsParticipantEcosystemKeeper) GetEcosystem(ctx context.Context, id uint64) (ectypes.Ecosystem, error) {
	return a.k.GetEcosystem(ctx, id)
}

func (a EcAsParticipantEcosystemKeeper) GetTrustUnitPrice(ctx sdk.Context) uint64 {
	return a.k.GetTrustUnitPrice(ctx)
}

// CoAsParticipantCorporationKeeper adapts x/co keeper to participanttypes.CorporationKeeper.
type CoAsParticipantCorporationKeeper struct {
	k cokeeper.Keeper
}

func NewCoAsParticipantCorporationKeeper(k cokeeper.Keeper) types.CorporationKeeper {
	return CoAsParticipantCorporationKeeper{k: k}
}

func (a CoAsParticipantCorporationKeeper) ResolveByPolicyAddress(ctx context.Context, policyAddress string) (types.CorporationView, bool) {
	coID, err := a.k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{Id: coID, PolicyAddress: policyAddress}, true
}

func (a CoAsParticipantCorporationKeeper) ResolveByID(ctx context.Context, id uint64) (types.CorporationView, bool) {
	co, err := a.k.Corporation.Get(ctx, id)
	if err != nil {
		return types.CorporationView{}, false
	}
	return types.CorporationView{Id: id, PolicyAddress: co.PolicyAddress}, true
}
