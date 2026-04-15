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
//
// TODO(spec-v4-draft-13 [AUTHZ-CHECK-1]): full spend-limit enforcement with
// period reset. The spec mandates:
//
//	"If oauthz.spend_limit is set, the remaining balance MUST be sufficient
//	 for the operation. After successful execution, the consumed amount MUST
//	 be deducted from the remaining balance. If oauthz.period is set and the
//	 current period has elapsed since the last reset, the remaining balance
//	 MUST be reset to oauthz.spend_limit before evaluating the check above."
//
// Implementing this requires: (a) a new OperatorAuthorizationUsage collection
// tracking (remaining_balance, last_reset) per (authority, operator), which
// needs a proto definition; (b) threading a spend_amount argument through
// every call site (currently none of tr/cs/perm/di/xr pass one); (c) updating
// remaining_balance on successful execution inside the outer msg handler.
// Until those arrive, the check only gates msg_type membership and expiration,
// matching pre-spec-v4 behavior. Tracked by the spec-alignment audit TODO.
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
