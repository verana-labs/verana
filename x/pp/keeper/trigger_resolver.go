package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/pp/types"
)

// TriggerResolver implements [MOD-PP-MSG-15]. It is event-only: it does not
// modify VPR state. It emits EventTypeTriggerResolver so off-chain trust
// resolvers (e.g. the verana-indexer) re-resolve the participant's did.
func (ms msgServer) TriggerResolver(goCtx context.Context, msg *types.MsgTriggerResolver) (*types.MsgTriggerResolverResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}

	// [MOD-PP-MSG-15-2-1] Basic checks.
	perm, err := ms.Keeper.GetParticipantByID(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("participant %d not found", msg.Id)
	}
	if err := IsValidParticipant(perm, now); err != nil {
		return nil, fmt.Errorf("participant %d is not active: %w", msg.Id, err)
	}

	// [AUTHZ-CHECK-5] Resolve the signing corporation account to its co.id.
	corpID, err := ms.corpIDFromAccount(ctx, msg.Corporation)
	if err != nil {
		return nil, err
	}

	// [MOD-PP-MSG-15-2-2] Authorization (at least one path MUST pass).
	if err := ms.authorizeTriggerResolver(ctx, msg.Corporation, corpID, msg.Operator, perm, now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// [MOD-PP-MSG-15-3] Execution: no state mutation, emit the event.
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTriggerResolver,
			sdk.NewAttribute(types.AttributeKeyParticipantID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyCorporation, msg.Corporation),
			sdk.NewAttribute(types.AttributeKeyOperator, msg.Operator),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	})

	return &types.MsgTriggerResolverResponse{}, nil
}

// authorizeTriggerResolver runs the two-path [MOD-PP-MSG-15-2-2] authorization.
// corpAccount is the signing corporation account, corpID its resolved co.id, perm
// the target Participant.
//
// The handler enforces the unconditional authorization: AUTHZ-CHECK-3 (Path 1)
// and AUTHZ-CHECK-1 (Path 2). The fee-grant checks (AUTHZ-CHECK-4 on Path 1,
// AUTHZ-CHECK-2 on Path 2) are conditional on the transaction fees being paid by
// the corporation account, which only the ante can determine; they are enforced
// there (issue #324 Phase 2).
func (ms msgServer) authorizeTriggerResolver(
	ctx sdk.Context, corpAccount string, corpID uint64, operator string, perm types.Participant, now time.Time,
) error {
	const msgType = types.MsgTriggerResolverTypeURL

	// Path 1 — vs_operator of the target participant (AUTHZ-CHECK-3).
	if corpID == perm.CorporationId && operator == perm.VsOperator {
		if err := ms.delegationKeeper.CheckVSOperatorAuthorizationOnParticipant(
			ctx, corpID, operator, perm.Id, msgType); err == nil {
			return nil
		}
	}

	// Path 2 — ancestor validator walk; the target perm itself is excluded
	// (AUTHZ-CHECK-1 for each active, same-corp ancestor).
	visited := map[uint64]bool{perm.Id: true}
	v := perm
	for v.ValidatorParticipantId != 0 && v.ValidatorParticipantId != perm.Id {
		next := v.ValidatorParticipantId
		if visited[next] {
			// Defensive: the participant tree is acyclic, but never loop forever.
			break
		}
		visited[next] = true

		parent, err := ms.Keeper.GetParticipantByID(ctx, next)
		if err != nil {
			break
		}
		v = parent

		if err := IsValidParticipant(v, now); err != nil {
			continue // skip inactive ancestor
		}
		if corpID != v.CorporationId {
			continue
		}
		if err := ms.delegationKeeper.CheckOperatorAuthorization(
			ctx, corpAccount, operator, msgType, now); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no authorization path matched for operator %s on participant %d", operator, perm.Id)
}
