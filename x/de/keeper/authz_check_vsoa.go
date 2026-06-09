package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/de/types"
)

// CheckVSOperatorAuthorizationOnParticipant implements [AUTHZ-CHECK-3]. The
// caller (a PP Msg handler) MUST resolve the signing corporation account to its
// co.id via AUTHZ-CHECK-5 first and pass corporationID (uint64), not the signing
// account.
//
// Steps:
//  1. A ParticipantAuthorizationRecord MUST exist for participantID.
//  2. The record MUST belong to VSOperatorAuthorization[corporationID, operator].
//  3. msgType MUST be in record.msg_types.
//  4. Cycle/expiration: if period is set and now >= expiration, reset the
//     remaining balances and roll expiration forward by period; else expiration
//     MUST be strictly in the future.
//  5. If spend_limit is set, remaining_spend MUST cover the operation and is
//     deducted after success. The keeper has no per-operation spend amount in
//     this context, so the amount-based deduction is deferred to the ante /
//     caller (matches the AUTHZ-CHECK-1 Check vs CheckWithSpend split).
func (k Keeper) CheckVSOperatorAuthorizationOnParticipant(
	ctx context.Context,
	corporationID uint64,
	operator string,
	participantID uint64,
	msgType string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// 1. Record MUST exist for participant_id.
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	// 2. Record MUST belong to VSOperatorAuthorization[corporationID, operator].
	if vsoa.CorporationId != corporationID || vsoa.VsOperator != operator {
		return types.ErrVSOperatorAuthzNotFound
	}

	idx := -1
	for i := range vsoa.Records {
		if vsoa.Records[i].ParticipantId == participantID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return types.ErrVSOperatorAuthzNotFound
	}
	rec := &vsoa.Records[idx]

	// 3. msg_type MUST be in record.msg_types.
	authorized := false
	for _, mt := range rec.MsgTypes {
		if mt == msgType {
			authorized = true
			break
		}
	}
	if !authorized {
		return fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgType)
	}

	// 4. Cycle / expiration.
	if rec.Period != nil && *rec.Period > 0 && rec.Expiration != nil && !rec.Expiration.After(now) {
		if len(rec.SpendLimit) > 0 {
			rec.RemainingSpend = rec.SpendLimit
		}
		if len(rec.FeeSpendLimit) > 0 {
			rec.RemainingFeeSpend = rec.FeeSpendLimit
		}
		newExp := now.Add(*rec.Period)
		rec.Expiration = &newExp
		if err := k.VSOperatorAuthorizations.Set(ctx, vsoaID, vsoa); err != nil {
			return fmt.Errorf("failed to persist cycle reset: %w", err)
		}
	} else if rec.Expiration == nil || !rec.Expiration.After(now) {
		return types.ErrAuthzExpired
	}

	// 5. spend_limit deduction is deferred to the ante / caller (no amount here).
	return nil
}

// CheckVSOperatorFeeGrant implements [AUTHZ-CHECK-4]. It uses the same record
// lookup as AUTHZ-CHECK-3.
//
//  1. record.with_feegrant MUST be true.
//  2. The cycle / expiration reset is handled by AUTHZ-CHECK-3 (run first).
//  3. If fee_spend_limit is set, remaining_fee_spend MUST cover the estimated tx
//     fees and is deducted after success. The keeper has no visibility into the
//     fee-payment mode or amount (ante context), so the amount-based check is
//     deferred to the ante handler (matches existing csps / trigger_resolver).
func (k Keeper) CheckVSOperatorFeeGrant(ctx context.Context, participantID uint64) error {
	vsoaID, err := k.VSOAByParticipant.Get(ctx, participantID)
	if err != nil {
		return types.ErrVSOperatorAuthzNotFound
	}
	vsoa, err := k.VSOperatorAuthorizations.Get(ctx, vsoaID)
	if err != nil {
		return fmt.Errorf("failed to load VSOperatorAuthorization %d: %w", vsoaID, err)
	}

	for i := range vsoa.Records {
		if vsoa.Records[i].ParticipantId == participantID {
			if !vsoa.Records[i].WithFeegrant {
				return types.ErrVSOFeegrantNotEnabled
			}
			// fee_spend_limit amount check deferred to ante handler.
			return nil
		}
	}
	return types.ErrVSOperatorAuthzNotFound
}
