package keeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// CheckOperatorAuthorization implements [AUTHZ-CHECK-1] without a spend amount.
// It verifies existence, expiration, and msg_type membership but does NOT touch
// the spend-limit ledger. Legacy entry point — callers with no meaningful spend
// amount continue to use this.
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

// CheckOperatorAuthorizationWithSpend implements the full [AUTHZ-CHECK-1]
// contract including the spend_limit / period-reset invariant. The spend ledger
// is keyed by the parent OperatorAuthorization id.
func (k Keeper) CheckOperatorAuthorizationWithSpend(
	ctx context.Context,
	corporation string,
	operator string,
	msgTypeURL string,
	now time.Time,
	spend sdk.Coins,
) error {
	oa, err := k.checkOperatorAuthorizationCore(ctx, corporation, operator, msgTypeURL, now)
	if err != nil {
		return err
	}
	if operator == "" {
		// Group-proposal path: nothing to meter.
		return nil
	}
	if len(oa.SpendLimit) == 0 || spend.IsZero() {
		// No spend limit configured, or caller isn't moving funds.
		return nil
	}

	// Load (or initialize) the usage record keyed by the OperatorAuthorization id.
	usage, err := k.OperatorAuthorizationUsage.Get(ctx, oa.Id)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return fmt.Errorf("failed to read usage ledger: %w", err)
		}
		// First use — seed remaining at spend_limit with last_reset = now.
		usage = types.OperatorAuthorizationUsage{
			OperatorAuthorizationId: oa.Id,
			Remaining:               oa.SpendLimit,
			LastReset:               now,
		}
	}

	// Period reset: if `period` is set and elapsed, refill remaining to spend_limit.
	if oa.Period != nil && *oa.Period > 0 {
		if now.Sub(usage.LastReset) >= *oa.Period {
			usage.Remaining = oa.SpendLimit
			usage.LastReset = now
		}
	}

	// Verify remaining balance covers the spend.
	if !usage.Remaining.IsAllGTE(spend) {
		return fmt.Errorf("%w: spend %s exceeds remaining %s",
			types.ErrAuthzSpendLimitExceeded, spend.String(), usage.Remaining.String())
	}

	// Debit remaining atomically with the check.
	usage.Remaining = usage.Remaining.Sub(spend...)
	if err := k.OperatorAuthorizationUsage.Set(ctx, oa.Id, usage); err != nil {
		return fmt.Errorf("failed to update authorization usage: %w", err)
	}

	return nil
}

// checkOperatorAuthorizationCore performs the expiration + msg_type checks and
// returns the loaded OperatorAuthorization so spend-limit enforcement can use it
// without a second keeper lookup.
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

	// 2. Check expiration: expired when now >= expiration.
	if oa.Expiration != nil && !oa.Expiration.After(now) {
		return types.OperatorAuthorization{}, types.ErrAuthzExpired
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
