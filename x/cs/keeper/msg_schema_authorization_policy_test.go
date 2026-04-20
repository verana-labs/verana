package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/cs/keeper"
	"github.com/verana-labs/verana/x/cs/types"
)

// validJsonSchemaForPolicy is a minimal valid JSON schema used in policy tests.
const validJsonSchemaForPolicy = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "PolicyTestSchema",
  "description": "Schema for authorization policy tests",
  "type": "object",
  "properties": {
    "name": { "type": "string" }
  },
  "required": ["name"],
  "additionalProperties": false
}`

func TestCreateSchemaAuthorizationPolicy_HappyPath(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_policy_________")).String()
	operator := sdk.AccAddress([]byte("oper_policy_________")).String()
	trID := mockTrk.CreateMockTrustRegistry(corporation, "did:example:happypath")

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	// Create credential schema
	createSchemaMsg := keeper.CreateMsgWithValidityPeriods(corporation, operator, trID, validJsonSchemaForPolicy, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	schemaResp, err := ms.CreateCredentialSchema(goCtx, createSchemaMsg)
	require.NoError(t, err)

	// [MOD-CS-MSG-5-3] Create schema authorization policy — effective_from/until are null at creation.
	msg := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Url:         "https://example.com/policy",
		DigestSri:   "sha256-abc123",
	}

	resp, err := ms.CreateSchemaAuthorizationPolicy(goCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotZero(t, resp.Id)

	// Verify stored policy
	policy, err := k.SchemaAuthorizationPolicies.Get(goCtx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, schemaResp.Id, policy.SchemaId)
	require.Equal(t, types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER, policy.Role)
	require.Equal(t, "https://example.com/policy", policy.Url)
	require.Equal(t, "sha256-abc123", policy.DigestSri)
	require.Equal(t, uint32(1), policy.Version)
	require.False(t, policy.Revoked)
	// Spec v4 draft 13: effective_from starts null (pending).
	require.Nil(t, policy.EffectiveFrom)
	require.Nil(t, policy.EffectiveUntil)
}

func TestCreateSchemaAuthorizationPolicy_VersionIncrement(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_policy2________")).String()
	operator := sdk.AccAddress([]byte("oper_policy2________")).String()
	trID := mockTrk.CreateMockTrustRegistry(corporation, "did:example:version-inc")

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	createSchemaMsg := keeper.CreateMsgWithValidityPeriods(corporation, operator, trID, validJsonSchemaForPolicy, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	schemaResp, err := ms.CreateCredentialSchema(goCtx, createSchemaMsg)
	require.NoError(t, err)

	// Create first policy
	msg1 := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Url:         "https://example.com/policy/v1",
		DigestSri:   "sha256-v1",
	}
	resp1, err := ms.CreateSchemaAuthorizationPolicy(goCtx, msg1)
	require.NoError(t, err)

	// Create second policy for the same (schema_id, role)
	msg2 := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Url:         "https://example.com/policy/v2",
		DigestSri:   "sha256-v2",
	}
	resp2, err := ms.CreateSchemaAuthorizationPolicy(goCtx, msg2)
	require.NoError(t, err)

	p1, _ := k.SchemaAuthorizationPolicies.Get(goCtx, resp1.Id)
	p2, _ := k.SchemaAuthorizationPolicies.Get(goCtx, resp2.Id)
	require.Equal(t, uint32(1), p1.Version)
	require.Equal(t, uint32(2), p2.Version)
}

func TestCreateSchemaAuthorizationPolicy_SchemaNotFound(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_notfound_______")).String()
	operator := sdk.AccAddress([]byte("oper_notfound_______")).String()
	_ = mockTrk

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	msg := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    9999, // non-existent schema
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Url:         "https://example.com/policy",
		DigestSri:   "sha256-abc",
	}

	resp, err := ms.CreateSchemaAuthorizationPolicy(goCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "credential schema not found")
	require.Nil(t, resp)

	_ = k
}

func TestCreateSchemaAuthorizationPolicy_WrongCorporation(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_wrong__________")).String()
	wrongCorp := sdk.AccAddress([]byte("wrong_corp__________")).String()
	operator := sdk.AccAddress([]byte("oper_wrong__________")).String()
	trID := mockTrk.CreateMockTrustRegistry(corporation, "did:example:wrongcorp")

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	createSchemaMsg := keeper.CreateMsgWithValidityPeriods(corporation, operator, trID, validJsonSchemaForPolicy, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	schemaResp, err := ms.CreateCredentialSchema(goCtx, createSchemaMsg)
	require.NoError(t, err)

	msg := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: wrongCorp, // wrong corporation
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Url:         "https://example.com/policy",
		DigestSri:   "sha256-abc",
	}

	resp, err := ms.CreateSchemaAuthorizationPolicy(goCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "corporation does not own the trust registry")
	require.Nil(t, resp)

	_ = k
}

func TestRevokeSchemaAuthorizationPolicy_HappyPath(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_revoke_________")).String()
	operator := sdk.AccAddress([]byte("oper_revoke_________")).String()
	trID := mockTrk.CreateMockTrustRegistry(corporation, "did:example:revoke")

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	createSchemaMsg := keeper.CreateMsgWithValidityPeriods(corporation, operator, trID, validJsonSchemaForPolicy, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	schemaResp, err := ms.CreateCredentialSchema(goCtx, createSchemaMsg)
	require.NoError(t, err)

	// [MOD-CS-MSG-5-3] Policy is created pending (effective_from null).
	createMsg := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Url:         "https://example.com/policy",
		DigestSri:   "sha256-abc",
	}
	policyResp, err := ms.CreateSchemaAuthorizationPolicy(goCtx, createMsg)
	require.NoError(t, err)

	policy, err := k.SchemaAuthorizationPolicies.Get(goCtx, policyResp.Id)
	require.NoError(t, err)
	require.Equal(t, uint32(1), policy.Version)

	// [MOD-CS-MSG-6] Activate the policy so it can be revoked per spec.
	_, err = ms.IncreaseActiveSchemaAuthorizationPolicyVersion(goCtx, &types.MsgIncreaseActiveSchemaAuthorizationPolicyVersion{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
	})
	require.NoError(t, err)

	// Revoke it
	revokeMsg := &types.MsgRevokeSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Version:     1,
	}
	revokeResp, err := ms.RevokeSchemaAuthorizationPolicy(goCtx, revokeMsg)
	require.NoError(t, err)
	require.NotNil(t, revokeResp)

	// Verify revoked
	policy, err = k.SchemaAuthorizationPolicies.Get(goCtx, policyResp.Id)
	require.NoError(t, err)
	require.True(t, policy.Revoked)
}

func TestRevokeSchemaAuthorizationPolicy_AlreadyRevoked(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_alrdy_revoked__")).String()
	operator := sdk.AccAddress([]byte("oper_alrdy_revoked__")).String()
	trID := mockTrk.CreateMockTrustRegistry(corporation, "did:example:already-revoked")

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	createSchemaMsg := keeper.CreateMsgWithValidityPeriods(corporation, operator, trID, validJsonSchemaForPolicy, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	schemaResp, err := ms.CreateCredentialSchema(goCtx, createSchemaMsg)
	require.NoError(t, err)

	// Create pending policy, then activate so revoke is valid.
	createMsg := &types.MsgCreateSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_VERIFIER,
		Url:         "https://example.com/policy",
		DigestSri:   "sha256-abc",
	}
	_, err = ms.CreateSchemaAuthorizationPolicy(goCtx, createMsg)
	require.NoError(t, err)

	_, err = ms.IncreaseActiveSchemaAuthorizationPolicyVersion(goCtx, &types.MsgIncreaseActiveSchemaAuthorizationPolicyVersion{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_VERIFIER,
	})
	require.NoError(t, err)

	revokeMsg := &types.MsgRevokeSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_VERIFIER,
		Version:     1,
	}

	// First revoke succeeds
	_, err = ms.RevokeSchemaAuthorizationPolicy(goCtx, revokeMsg)
	require.NoError(t, err)

	// Second revoke must fail with already revoked
	_, err = ms.RevokeSchemaAuthorizationPolicy(goCtx, revokeMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already revoked")

	_ = k
}

func TestRevokeSchemaAuthorizationPolicy_NotFound(t *testing.T) {
	k, mockTrk, rawCtx := keepertest.CredentialschemaKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	corporation := sdk.AccAddress([]byte("corp_notfound2______")).String()
	operator := sdk.AccAddress([]byte("oper_notfound2______")).String()
	trID := mockTrk.CreateMockTrustRegistry(corporation, "did:example:notfound2")

	now := time.Now().UTC()
	sdkCtx := sdk.UnwrapSDKContext(rawCtx).WithBlockTime(now)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	createSchemaMsg := keeper.CreateMsgWithValidityPeriods(corporation, operator, trID, validJsonSchemaForPolicy, 365, 365, 180, 180, 180, 2, 2, 2, 1, "tu", "sha256")
	schemaResp, err := ms.CreateCredentialSchema(goCtx, createSchemaMsg)
	require.NoError(t, err)

	// Try to revoke version 99 which does not exist
	revokeMsg := &types.MsgRevokeSchemaAuthorizationPolicy{
		Corporation: corporation,
		Operator:    operator,
		SchemaId:    schemaResp.Id,
		Role:        types.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER,
		Version:     99,
	}

	resp, err := ms.RevokeSchemaAuthorizationPolicy(goCtx, revokeMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no policy found")
	require.Nil(t, resp)

	_ = k
}
