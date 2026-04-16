package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/verana-labs/verana/x/cs/types"
)

// getSchemaAuthPoliciesForRole returns all policies for (schema_id, role).
func (k Keeper) getSchemaAuthPoliciesForRole(ctx sdk.Context, schemaID uint64, role types.SchemaAuthorizationPolicyRole) ([]types.SchemaAuthorizationPolicy, error) {
	var policies []types.SchemaAuthorizationPolicy
	err := k.SchemaAuthorizationPolicies.Walk(ctx, nil, func(_ uint64, p types.SchemaAuthorizationPolicy) (bool, error) {
		if p.SchemaId == schemaID && p.Role == role {
			policies = append(policies, p)
		}
		return false, nil
	})
	return policies, err
}

// [MOD-CS-MSG-5] CreateSchemaAuthorizationPolicy
func (ms msgServer) CreateSchemaAuthorizationPolicy(goCtx context.Context, msg *types.MsgCreateSchemaAuthorizationPolicy) (*types.MsgCreateSchemaAuthorizationPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// AUTHZ-CHECK
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.cs.v1.MsgCreateSchemaAuthorizationPolicy", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// effective_from must not be in the past
	if msg.EffectiveFrom.Before(now) {
		return nil, fmt.Errorf("effective_from must not be in the past")
	}

	// Load credential schema
	cs, err := ms.CredentialSchema.Get(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}

	// Check ownership via trust registry
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Corporation != msg.Corporation {
		return nil, fmt.Errorf("corporation does not own the trust registry for this credential schema")
	}

	// Determine next version for this (schema_id, role) pair
	existing, err := ms.getSchemaAuthPoliciesForRole(ctx, msg.SchemaId, msg.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing policies: %w", err)
	}
	nextVersion := uint32(len(existing) + 1)

	id, err := ms.GetNextID(ctx, types.CounterKeySchemaAuthorizationPolicy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate policy id: %w", err)
	}

	policy := types.SchemaAuthorizationPolicy{
		Id:             id,
		SchemaId:       msg.SchemaId,
		Role:           msg.Role,
		Url:            msg.Url,
		DigestSri:      msg.DigestSri,
		EffectiveFrom:  msg.EffectiveFrom,
		EffectiveUntil: msg.EffectiveUntil,
		Revoked:        false,
		Created:        now,
		Version:        nextVersion,
	}

	if err := ms.SchemaAuthorizationPolicies.Set(ctx, id, policy); err != nil {
		return nil, fmt.Errorf("failed to store policy: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"create_schema_authorization_policy",
		sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
		sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
		sdk.NewAttribute("role", msg.Role.String()),
		sdk.NewAttribute("version", strconv.FormatUint(uint64(nextVersion), 10)),
	))

	return &types.MsgCreateSchemaAuthorizationPolicyResponse{Id: id}, nil
}

// [MOD-CS-MSG-6] IncreaseActiveSchemaAuthorizationPolicyVersion
func (ms msgServer) IncreaseActiveSchemaAuthorizationPolicyVersion(goCtx context.Context, msg *types.MsgIncreaseActiveSchemaAuthorizationPolicyVersion) (*types.MsgIncreaseActiveSchemaAuthorizationPolicyVersionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// AUTHZ-CHECK
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.cs.v1.MsgIncreaseActiveSchemaAuthorizationPolicyVersion", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Load credential schema and check ownership
	cs, err := ms.CredentialSchema.Get(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Corporation != msg.Corporation {
		return nil, fmt.Errorf("corporation does not own the trust registry for this credential schema")
	}

	// Get all policies for this (schema_id, role)
	policies, err := ms.getSchemaAuthPoliciesForRole(ctx, msg.SchemaId, msg.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	if len(policies) == 0 {
		return nil, fmt.Errorf("no schema authorization policy exists for schema_id %d and role %s", msg.SchemaId, msg.Role)
	}

	// Find pending policies (not revoked, effective_from in the future)
	var pendingPolicies []types.SchemaAuthorizationPolicy
	for _, p := range policies {
		if p.Revoked {
			continue
		}
		if p.EffectiveFrom.After(now) {
			pendingPolicies = append(pendingPolicies, p)
		}
	}

	if len(pendingPolicies) == 0 {
		return nil, fmt.Errorf("no future (non-active) policy version exists for schema_id %d and role %s", msg.SchemaId, msg.Role)
	}

	// Activate the lowest-version pending policy by setting its effective_from to now
	next := pendingPolicies[0]
	for _, p := range pendingPolicies[1:] {
		if p.Version < next.Version {
			next = p
		}
	}

	next.EffectiveFrom = now
	if err := ms.SchemaAuthorizationPolicies.Set(ctx, next.Id, next); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"increase_active_schema_authorization_policy_version",
		sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
		sdk.NewAttribute("role", msg.Role.String()),
		sdk.NewAttribute("new_active_version", strconv.FormatUint(uint64(next.Version), 10)),
	))

	return &types.MsgIncreaseActiveSchemaAuthorizationPolicyVersionResponse{}, nil
}

// [MOD-CS-MSG-7] RevokeSchemaAuthorizationPolicy
func (ms msgServer) RevokeSchemaAuthorizationPolicy(goCtx context.Context, msg *types.MsgRevokeSchemaAuthorizationPolicy) (*types.MsgRevokeSchemaAuthorizationPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// AUTHZ-CHECK
	if ms.delegationKeeper == nil {
		return nil, fmt.Errorf("delegation keeper is required for operator authorization")
	}
	if err := ms.delegationKeeper.CheckOperatorAuthorization(ctx, msg.Corporation, msg.Operator, "/verana.cs.v1.MsgRevokeSchemaAuthorizationPolicy", now); err != nil {
		return nil, fmt.Errorf("authorization check failed: %w", err)
	}

	// Load credential schema and check ownership
	cs, err := ms.CredentialSchema.Get(ctx, msg.SchemaId)
	if err != nil {
		return nil, fmt.Errorf("credential schema not found: %w", err)
	}
	tr, err := ms.trustRegistryKeeper.GetTrustRegistry(ctx, cs.TrId)
	if err != nil {
		return nil, fmt.Errorf("trust registry not found: %w", err)
	}
	if tr.Corporation != msg.Corporation {
		return nil, fmt.Errorf("corporation does not own the trust registry for this credential schema")
	}

	// Find the policy for (schema_id, role, version)
	policies, err := ms.getSchemaAuthPoliciesForRole(ctx, msg.SchemaId, msg.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}

	var target *types.SchemaAuthorizationPolicy
	for i := range policies {
		if policies[i].Version == msg.Version {
			target = &policies[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("no policy found for schema_id %d, role %s, version %d", msg.SchemaId, msg.Role, msg.Version)
	}
	if target.Revoked {
		return nil, fmt.Errorf("policy is already revoked")
	}
	// Policy must be active (effective_from <= now) to be revoked
	if target.EffectiveFrom.After(now) {
		return nil, fmt.Errorf("policy is not yet active; cannot revoke a future policy")
	}

	target.Revoked = true
	if err := ms.SchemaAuthorizationPolicies.Set(ctx, target.Id, *target); err != nil {
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"revoke_schema_authorization_policy",
		sdk.NewAttribute("schema_id", strconv.FormatUint(msg.SchemaId, 10)),
		sdk.NewAttribute("role", msg.Role.String()),
		sdk.NewAttribute("version", strconv.FormatUint(uint64(msg.Version), 10)),
	))

	return &types.MsgRevokeSchemaAuthorizationPolicyResponse{}, nil
}
