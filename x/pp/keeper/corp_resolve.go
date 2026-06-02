package keeper

import (
	"context"
	"fmt"
)

// corpIDFromAccount resolves a signing corporation account (policy_address) to
// its uint64 Corporation id (AUTHZ-CHECK-5 subject resolution). The Participant
// entity persists corporation_id; callers pass the account from the Msg.
func (ms msgServer) corpIDFromAccount(ctx context.Context, account string) (uint64, error) {
	co, ok := ms.coKeeper.ResolveByPolicyAddress(ctx, account)
	if !ok {
		return 0, fmt.Errorf("signing corporation not registered: %s", account)
	}
	return co.Id, nil
}

// corpAccountFromID resolves a Participant's corporation_id back to the
// Corporation policy_address account used for fund-flows (trust deposit,
// feegrant, slashing).
func (ms msgServer) corpAccountFromID(ctx context.Context, id uint64) (string, error) {
	co, ok := ms.coKeeper.ResolveByID(ctx, id)
	if !ok {
		return "", fmt.Errorf("corporation %d not found", id)
	}
	return co.PolicyAddress, nil
}
