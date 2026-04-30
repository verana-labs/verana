package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/perm/types"
)

// triggerResolverMsgTypeURL is the canonical Msg type URL used by AUTHZ-CHECK
// when authorization falls through to Path 2b (regular OperatorAuthorization
// on an ancestor).
const triggerResolverMsgTypeURL = "/verana.perm.v1.MsgTriggerResolver"

// TriggerResolver implements [MOD-PERM-MSG-15] Trigger Resolver.
//
// It emits an on-chain event signaling that an external trust resolver should
// re-resolve the DID registered in the target permission. The VPR makes no
// state changes.
//
// Authorization succeeds if EITHER of these paths passes:
//
//   Path 1 — target perm vs_operator:
//     msg.corporation == perm.corporation
//     msg.operator    == perm.vs_operator
//     AUTHZ-CHECK-3 on (corporation, operator, perm)
//
//   Path 2 — ancestor validator walk (target perm itself excluded):
//     For any ACTIVE ancestor v in the validator_perm_id chain where
//     msg.corporation == v.corporation, EITHER:
//       2a: msg.operator == v.vs_operator AND AUTHZ-CHECK-3 on (corporation, operator, v)
//       2b: AUTHZ-CHECK on (corporation, operator) for this Msg type
//
// AUTHZ-CHECK-3 here implements sub-checks (a)(b)(c) only. Sub-checks (d)(e),
// and AUTHZ-CHECK-4 (a)(b)(c), require remaining-balance / last-reset state
// that the codebase does not yet model and consume nothing for TriggerResolver
// (no spend_limit decrement, no fee_spend_limit decrement). They are tracked
// as a separate compliance task; see the TODOs below.
func (ms msgServer) TriggerResolver(goCtx context.Context, msg *types.MsgTriggerResolver) (*types.MsgTriggerResolverResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for trigger resolver authorization")
	}

	// Preconditions: load perm, active, did present.
	perm, err := ms.Permission.Get(ctx, msg.Id)
	if err != nil {
		return nil, fmt.Errorf("permission %d not found: %w", msg.Id, err)
	}
	if err := IsValidPermission(perm, now); err != nil {
		return nil, errorsmod.Wrapf(types.ErrPermissionNotActive, "%v", err)
	}
	if perm.Did == "" {
		return nil, types.ErrPermissionDIDEmpty
	}

	// Path 1: target perm vs_operator.
	if msg.Corporation == perm.Corporation && msg.Operator == perm.VsOperator {
		if err := ms.checkVSOperatorAuthorizationOnPerm(ctx, msg.Corporation, msg.Operator, perm); err == nil {
			return ms.emitTriggerResolverEvent(ctx, msg.Id)
		}
	}

	// Path 2: ancestor validator walk (target excluded).
	if ms.tryAncestorAuthorization(ctx, now, msg, perm) {
		return ms.emitTriggerResolverEvent(ctx, msg.Id)
	}

	return nil, errorsmod.Wrapf(
		types.ErrTriggerResolverUnauthorized,
		"corporation=%s operator=%s perm_id=%d",
		msg.Corporation, msg.Operator, msg.Id,
	)
}

// tryAncestorAuthorization walks perm.validator_perm_id upward and returns true
// as soon as Path 2a or Path 2b authorizes the message. The starting perm is
// never considered an ancestor of itself, so a "self-match as ancestor" cannot
// pass authorization.
func (ms msgServer) tryAncestorAuthorization(
	ctx sdk.Context,
	now time.Time,
	msg *types.MsgTriggerResolver,
	target types.Permission,
) bool {
	visited := make(map[uint64]struct{}, 8)
	cur := target.ValidatorPermId
	for cur != 0 {
		if _, seen := visited[cur]; seen {
			// Defensive: a cycle in validator_perm_id should never exist, but
			// if state is corrupted we abort the walk rather than spin.
			return false
		}
		visited[cur] = struct{}{}

		v, err := ms.Permission.Get(ctx, cur)
		if err != nil {
			return false
		}
		if IsValidPermission(v, now) == nil && msg.Corporation == v.Corporation {
			// 2a: VSOA on ancestor.
			if msg.Operator == v.VsOperator {
				if err := ms.checkVSOperatorAuthorizationOnPerm(ctx, msg.Corporation, msg.Operator, v); err == nil {
					return true
				}
			}
			// 2b: regular OperatorAuthorization for this msg type.
			if err := ms.delegationKeeper.CheckOperatorAuthorization(
				ctx,
				msg.Corporation,
				msg.Operator,
				triggerResolverMsgTypeURL,
				now,
			); err == nil {
				return true
			}
		}
		cur = v.ValidatorPermId
	}
	return false
}

// checkVSOperatorAuthorizationOnPerm enforces AUTHZ-CHECK-3 sub-checks (a)(b)(c)
// for a (corporation, operator, perm) tuple:
//
//	(a) VSOperatorAuthorization vso exists for (corporation, operator)
//	(b) perm.id is in vso.permissions
//	(c) perm.vs_operator_authz_enabled == true
//
// TODO(spec-compliance, MOD-PERM-MSG-10): AUTHZ-CHECK-3 (d) period reset and
// (e) spend_limit sufficiency + decrement are not implemented here. They are
// inert for TriggerResolver (consumes no spend_limit) and not yet implemented
// for CSPS either; tracking issue should land alongside the new
// initial_*/last_reset_at proto fields.
//
// TODO(spec-compliance, MOD-PERM-MSG-10): AUTHZ-CHECK-4 (a)(b)(c) — feegrant-
// path checks rely on ante-handler context (whether the tx fee was paid via
// fee grant) which is not available here. Skipped in keeper for parity with
// csps.go; defer to ante / feegrant module.
func (ms msgServer) checkVSOperatorAuthorizationOnPerm(
	ctx context.Context,
	corporation, operator string,
	perm types.Permission,
) error {
	// (a) VSOA exists for (corporation, operator).
	if err := ms.delegationKeeper.CheckVSOperatorAuthorization(ctx, corporation, operator); err != nil {
		return fmt.Errorf("VS operator authorization check failed: %w", err)
	}
	// (b) perm.id is in vso.permissions.
	permIDs, err := ms.delegationKeeper.GetVSOAPermissions(ctx, corporation, operator)
	if err != nil {
		return fmt.Errorf("failed to load VSOA permissions: %w", err)
	}
	found := false
	for _, id := range permIDs {
		if id == perm.Id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("permission %d is not in VSOA permission list for (%s, %s)", perm.Id, corporation, operator)
	}
	// (c) perm.vs_operator_authz_enabled is true.
	if !perm.VsOperatorAuthzEnabled {
		return fmt.Errorf("vs_operator_authz_enabled is false on permission %d", perm.Id)
	}
	return nil
}

func (ms msgServer) emitTriggerResolverEvent(ctx sdk.Context, permID uint64) (*types.MsgTriggerResolverResponse, error) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTriggerResolver,
			sdk.NewAttribute(types.AttributeKeyPermissionID, strconv.FormatUint(permID, 10)),
		),
	})
	return &types.MsgTriggerResolverResponse{}, nil
}
