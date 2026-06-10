package keeper

import (
	"context"
	"fmt"
	"time"

	"github.com/verana-labs/verana/x/de/types"
)

// CheckOperatorAuthorization implements [AUTHZ-CHECK-1] at the membership level:
// it verifies existence, period auto-renewal / expiration, and msg_type
// membership. The per-operation spend debit (AUTHZ-CHECK-1 step 3) is enforced in
// the ante (issue #324 Phase 2), so this entry point does not take a spend amount.
//
// If operator is empty, the corporation is acting alone (e.g. via group
// proposal) and the check is skipped.
//
// The `corporation` argument is the signing corporation account (policy_address)
// and is resolved to its co.id via AUTHZ-CHECK-5 before the (corporation_id,
// operator) index lookup.
func (k Keeper) CheckOperatorAuthorization(
	ctx context.Context,
	corporation string,
	operator string,
	msgTypeURL string,
	now time.Time,
) error {
	_, err := k.checkOperatorAuthorizationCore(ctx, corporation, operator, msgTypeURL, now)
	return err
}

// checkOperatorAuthorizationCore performs the period-renewal / expiration +
// msg_type checks and returns the loaded OperatorAuthorization.
func (k Keeper) checkOperatorAuthorizationCore(
	ctx context.Context,
	corporation string,
	operator string,
	msgTypeURL string,
	now time.Time,
) (types.OperatorAuthorization, error) {
	// If operator is empty, the corporation is acting alone (group proposal) — skip.
	if operator == "" {
		return types.OperatorAuthorization{}, nil
	}

	// Resolve the signing corporation account to its co.id (AUTHZ-CHECK-5). An
	// unregistered corporation cannot have granted any authorization.
	co, err := k.corporationKeeper().ResolveCorporationByPolicyAddress(ctx, corporation)
	if err != nil {
		return types.OperatorAuthorization{}, types.ErrAuthzNotFound
	}

	// 1. Load OperatorAuthorization via the (corporation_id, operator) index.
	oa, found, err := k.getOperatorAuthorizationByCorpOp(ctx, co.Id, operator)
	if err != nil {
		return types.OperatorAuthorization{}, err
	}
	if !found {
		return types.OperatorAuthorization{}, types.ErrAuthzNotFound
	}

	// 2. Expiration / period auto-renewal (AUTHZ-CHECK-1 step 2).
	if oa.Expiration != nil {
		if oa.Period != nil && *oa.Period > 0 && !oa.Expiration.After(now) {
			// Period elapsed: reset runtime balances and roll expiration forward.
			if len(oa.SpendLimit) > 0 {
				oa.RemainingSpend = oa.SpendLimit
			}
			if len(oa.FeeSpendLimit) > 0 {
				oa.RemainingFeeSpend = oa.FeeSpendLimit
			}
			newExp := now.Add(*oa.Period)
			oa.Expiration = &newExp
			if err := k.OperatorAuthorizations.Set(ctx, oa.Id, oa); err != nil {
				return types.OperatorAuthorization{}, fmt.Errorf("failed to persist authz renewal: %w", err)
			}
		} else if !oa.Expiration.After(now) {
			return types.OperatorAuthorization{}, types.ErrAuthzExpired
		}
	}

	// 3. Check that the requested msg type is authorized.
	found = false
	for _, mt := range oa.MsgTypes {
		if mt == msgTypeURL {
			found = true
			break
		}
	}
	if !found {
		return types.OperatorAuthorization{}, fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgTypeURL)
	}

	return oa, nil
}
