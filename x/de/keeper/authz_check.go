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

// CheckOperatorAuthorization implements [AUTHZ-CHECK-1] without a spend
// amount. It verifies existence, expiration, and msg_type membership but does
// NOT touch the spend-limit ledger. Legacy entry point — callers that have
// no meaningful spend amount (e.g. pure control-plane messages like
// MsgArchiveTrustRegistry) continue to use this.
//
// If operator is empty, the corporation is acting alone (e.g. via group
// proposal) and the check is skipped.
//
// Checks performed:
//  1. OperatorAuthorization must exist for (corporation, operator)
//  2. If expiration is set, it must be in the future
//  3. The requested msgTypeURL must be in the authorization's msg_types
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
// contract including the spend_limit / period-reset invariant:
//
//	"If oauthz.spend_limit is set, the remaining balance MUST be sufficient
//	 for the operation. After successful execution, the consumed amount MUST
//	 be deducted from the remaining balance. If oauthz.period is set and the
//	 current period has elapsed since the last reset, the remaining balance
//	 MUST be reset to oauthz.spend_limit before evaluating the check above."
//
// Callers that actually consume funds (bank transfers, trust deposit
// adjustments) should use this variant and pass the total coin amount they
// intend to move. The remaining balance is decremented atomically with the
// check, so callers only need to invoke it once per (authority, operator,
// msg type) tuple.
//
// When spend is zero or the authorization has no spend_limit configured, the
// extra ledger work is skipped and behavior matches CheckOperatorAuthorization.
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

	// Load (or initialize) the usage record for this (corporation, operator) tuple.
	key := collections.Join(corporation, operator)
	usage, err := k.OperatorAuthorizationUsage.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return fmt.Errorf("failed to read usage ledger: %w", err)
		}
		// First use — seed remaining at spend_limit with last_reset = now.
		usage = types.OperatorAuthorizationUsage{
			Corporation: corporation,
			Operator:    operator,
			Remaining:   oa.SpendLimit,
			LastReset:   now,
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
	if err := k.OperatorAuthorizationUsage.Set(ctx, key, usage); err != nil {
		return fmt.Errorf("failed to update authorization usage: %w", err)
	}

	return nil
}

// checkOperatorAuthorizationCore performs the expiration + msg_type checks
// and returns the loaded OperatorAuthorization so spend-limit enforcement can
// use it without a second keeper lookup.
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

	// 1. Load OperatorAuthorization
	key := collections.Join(corporation, operator)
	oa, err := k.OperatorAuthorizations.Get(ctx, key)
	if err != nil {
		return types.OperatorAuthorization{}, types.ErrAuthzNotFound
	}

	// 2. Check expiration: expired when now >= expiration (i.e. expiration is not strictly after now).
	if oa.Expiration != nil && !oa.Expiration.After(now) {
		return types.OperatorAuthorization{}, types.ErrAuthzExpired
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
		return types.OperatorAuthorization{}, fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgTypeURL)
	}

	return oa, nil
}
