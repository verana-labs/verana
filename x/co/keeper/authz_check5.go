package keeper

import (
	"context"

	"cosmossdk.io/errors"

	"github.com/verana-labs/verana/x/co/types"
)

// ResolveCorporationByPolicyAddress implements AUTHZ-CHECK-5 (Corporation
// Registration check): it resolves the signing `corporation` account
// (= policy_address) to its registered Corporation entry. If no Corporation is
// bound to that policy_address it returns ErrCorporationNotRegistered, which
// references MOD-CO-MSG-1 and the offending policy_address. On success the
// resolved co.id is the corporation_id the calling message MUST use downstream.
//
// This is the single canonical resolver every delegable Msg routes through
// (directly for MOD-CO, or via a per-module CorporationKeeper adapter).
func (k Keeper) ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (types.Corporation, error) {
	id, err := k.CorporationByPolicyAddr.Get(ctx, policyAddress)
	if err != nil {
		return types.Corporation{}, errors.Wrapf(types.ErrCorporationNotRegistered,
			"signing account %s has not been registered as the policy_address of a Corporation (see MOD-CO-MSG-1)", policyAddress)
	}
	return k.Corporation.Get(ctx, id)
}
