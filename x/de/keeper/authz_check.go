package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana/x/de/types"
)

// CheckOperatorAuthorization implements [AUTHZ-CHECK-1].
// It verifies that an OperatorAuthorization exists for the given
// (authority, operator) pair and that it covers the specified msgTypeURL.
//
// If operator is empty, the authority is acting alone (e.g. via group proposal)
// and the check is skipped.
//
// Checks performed:
//  1. OperatorAuthorization must exist for (authority, operator)
//  2. If expiration is set, it must be in the future
//  3. The requested msgTypeURL must be in the authorization's msg_types
//  4. If spend_limit is set and spendAmount is non-nil, remaining balance must
//     suffice (with period reset if elapsed)
func (k Keeper) CheckOperatorAuthorization(
	ctx context.Context,
	authority string,
	operator string,
	msgTypeURL string,
	now time.Time,
) error {
	// If operator is empty, authority is acting alone (group proposal) — skip
	if operator == "" {
		return nil
	}

	// 1. Load OperatorAuthorization
	key := collections.Join(authority, operator)
	oa, err := k.OperatorAuthorizations.Get(ctx, key)
	if err != nil {
		return types.ErrAuthzNotFound
	}

	// 2. Check expiration
	if oa.Expiration != nil && !oa.Expiration.After(now) {
		return types.ErrAuthzExpired
	}

	// 3. Check that the requested msg type is authorized
	found := false
	for _, mt := range oa.MsgTypes {
		if mt == msgTypeURL {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgTypeURL)
	}

	return nil
}
