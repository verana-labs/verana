package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/de/keeper"
	"github.com/verana-labs/verana/x/de/types"
)

func setupMsgServer(t *testing.T) (keeper.Keeper, types.MsgServer, sdk.Context) {
	t.Helper()
	f := initFixture(t)
	ctx := sdk.UnwrapSDKContext(f.ctx)
	return f.keeper, keeper.NewMsgServerImpl(f.keeper), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

// --- Test address conventions ---
//
// In the VPR spec, "authority" is a group account (x/group policy address).
// At the unit-test level we use plain sdk.AccAddress values to represent these
// addresses — the DE module never calls the group keeper, it only receives
// messages after the group proposal has been executed by the Cosmos SDK runtime.
//
// The three invocation patterns for MOD-DE-MSG-3 / MOD-DE-MSG-4:
//
//   1. Group proposal  – authority signs via group governance; operator is "".
//                         CheckOperatorAuthorization skips the AUTHZ check.
//   2. Authority + op  – an existing operator (who was previously onboarded)
//                         executes on behalf of the authority; AUTHZ is verified.
//   3. Module call     – internal keeper methods (MOD-DE-MSG-1, MOD-DE-MSG-2)
//                         called directly, no msg-server routing.
//
// The first operator is always bootstrapped via pattern 1 (group proposal).

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-1] GrantFeeAllowance (internal keeper method)
// ---------------------------------------------------------------------------

func TestGrantFeeAllowance(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	// authority represents a group policy account (see conventions above)
	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	futureExp := ctx.BlockTime().Add(24 * time.Hour)
	pastExp := ctx.BlockTime().Add(-1 * time.Hour)
	validSpendLimit := sdk.NewCoins(sdk.NewInt64Coin("stake", 1000))
	invalidSpendLimit := sdk.Coins{sdk.Coin{Denom: "", Amount: sdkmath.NewInt(-1)}}
	validPeriod := time.Hour
	invalidPeriod := -time.Hour
	zeroPeriod := time.Duration(0)
	nowExp := ctx.BlockTime()

	testCases := []struct {
		name       string
		authority  string
		grantee   string
		msgTypes   []string
		expiration *time.Time
		spendLimit sdk.Coins
		period     *time.Duration
		expectErr  bool
		errContains string
	}{
		{
			name:      "Valid: basic grant",
			authority: authority,
			grantee:   grantee,
			msgTypes:  validMsgTypes,
			expectErr: false,
		},
		{
			name:       "Valid: with expiration",
			authority:  authority,
			grantee:    grantee,
			msgTypes:   validMsgTypes,
			expiration: &futureExp,
			expectErr:  false,
		},
		{
			name:       "Valid: with spend limit and period",
			authority:  authority,
			grantee:    grantee,
			msgTypes:   validMsgTypes,
			spendLimit: validSpendLimit,
			period:     &validPeriod,
			expectErr:  false,
		},
		{
			name:       "Valid: with all optional fields",
			authority:  authority,
			grantee:    grantee,
			msgTypes:   validMsgTypes,
			expiration: &futureExp,
			spendLimit: validSpendLimit,
			period:     &validPeriod,
			expectErr:  false,
		},
		{
			name:        "Invalid: empty msg_types",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    []string{},
			expectErr:   true,
			errContains: "msg_types must not be empty",
		},
		{
			name:        "Invalid: non-VPR delegable msg type",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    []string{"/cosmos.bank.v1beta1.MsgSend"},
			expectErr:   true,
			errContains: "invalid or non-delegable message type",
		},
		{
			name:        "Invalid: mix of valid and invalid msg types",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    []string{"/verana.tr.v1.MsgCreateTrustRegistry", "/cosmos.bank.v1beta1.MsgSend"},
			expectErr:   true,
			errContains: "invalid or non-delegable message type",
		},
		{
			name:        "Invalid: expiration in the past",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    validMsgTypes,
			expiration:  &pastExp,
			expectErr:   true,
			errContains: "expiration must be in the future",
		},
		{
			name:        "Invalid: bad spend limit",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    validMsgTypes,
			spendLimit:  invalidSpendLimit,
			expectErr:   true,
			errContains: "invalid spend limit",
		},
		{
			name:        "Invalid: negative period",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    validMsgTypes,
			period:      &invalidPeriod,
			expectErr:   true,
			errContains: "period must be a positive duration",
		},
		{
			name:      "Valid: multiple VPR delegable msg types",
			authority: authority,
			grantee:   grantee,
			msgTypes: []string{
				"/verana.tr.v1.MsgCreateTrustRegistry",
				"/verana.cs.v1.MsgCreateCredentialSchema",
				"/verana.perm.v1.MsgCreatePermission",
			},
			expectErr: false,
		},
		{
			name:        "Invalid: zero period",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    validMsgTypes,
			period:      &zeroPeriod,
			expectErr:   true,
			errContains: "period must be a positive duration",
		},
		{
			name:        "Invalid: expiration exactly at block time (boundary)",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    validMsgTypes,
			expiration:  &nowExp,
			expectErr:   true,
			errContains: "expiration must be in the future",
		},
		{
			name:        "Invalid: CreateOrUpdatePermissionSession excluded",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    []string{"/verana.perm.v1.MsgCreateOrUpdatePermissionSession"},
			expectErr:   true,
			errContains: "invalid or non-delegable message type",
		},
		{
			name:        "Invalid: UpdateParams excluded",
			authority:   authority,
			grantee:     grantee,
			msgTypes:    []string{"/verana.de.v1.MsgUpdateParams"},
			expectErr:   true,
			errContains: "invalid or non-delegable message type",
		},
		{
			name:       "Valid: spend limit without period",
			authority:  authority,
			grantee:    grantee,
			msgTypes:   validMsgTypes,
			spendLimit: validSpendLimit,
			expectErr:  false,
		},
		{
			name:      "Valid: period without spend limit",
			authority: authority,
			grantee:   grantee,
			msgTypes:  validMsgTypes,
			period:    &validPeriod,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := k.GrantFeeAllowance(ctx, tc.authority, tc.grantee, tc.msgTypes, tc.expiration, tc.spendLimit, tc.period)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)

				// Verify FeeGrant was stored
				key := collections.Join(tc.authority, tc.grantee)
				fg, err := k.FeeGrants.Get(ctx, key)
				require.NoError(t, err)
				require.Equal(t, tc.authority, fg.Grantor)
				require.Equal(t, tc.grantee, fg.Grantee)
				require.Equal(t, tc.msgTypes, fg.MsgTypes)
			}
		})
	}
}

func TestGrantFeeAllowance_UpdateExisting(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	msgTypes1 := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	msgTypes2 := []string{"/verana.cs.v1.MsgCreateCredentialSchema"}

	// Create initial grant
	err := k.GrantFeeAllowance(ctx, authority, grantee, msgTypes1, nil, nil, nil)
	require.NoError(t, err)

	// Verify initial state
	key := collections.Join(authority, grantee)
	fg, err := k.FeeGrants.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, msgTypes1, fg.MsgTypes)

	// Update with new msg types
	err = k.GrantFeeAllowance(ctx, authority, grantee, msgTypes2, nil, nil, nil)
	require.NoError(t, err)

	// Verify updated state
	fg, err = k.FeeGrants.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, msgTypes2, fg.MsgTypes)
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-2] RevokeFeeAllowance (internal keeper method)
// ---------------------------------------------------------------------------

func TestRevokeFeeAllowance(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()

	testCases := []struct {
		name      string
		setupFunc func()
		authority string
		grantee   string
		expectErr bool
		errContains string
		expectRemoved bool
	}{
		{
			name:      "Valid: revoke existing grant",
			authority: authority,
			grantee:   grantee,
			setupFunc: func() {
				err := k.GrantFeeAllowance(ctx, authority, grantee,
					[]string{"/verana.tr.v1.MsgCreateTrustRegistry"}, nil, nil, nil)
				require.NoError(t, err)
			},
			expectErr:     false,
			expectRemoved: true,
		},
		{
			name:          "Valid: no-op when grant does not exist",
			authority:     authority,
			grantee:       grantee,
			expectErr:     false,
			expectRemoved: false,
		},
		{
			name:        "Invalid: empty authority",
			authority:   "",
			grantee:     grantee,
			expectErr:   true,
			errContains: "authority must be specified",
		},
		{
			name:        "Invalid: empty grantee",
			authority:   authority,
			grantee:     "",
			expectErr:   true,
			errContains: "grantee must be specified",
		},
		{
			name:        "Invalid: both empty",
			authority:   "",
			grantee:     "",
			expectErr:   true,
			errContains: "authority must be specified",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			err := k.RevokeFeeAllowance(ctx, tc.authority, tc.grantee)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)

				if tc.expectRemoved {
					// Verify FeeGrant was removed
					key := collections.Join(tc.authority, tc.grantee)
					has, err := k.FeeGrants.Has(ctx, key)
					require.NoError(t, err)
					require.False(t, has)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-3] GrantOperatorAuthorization
// ---------------------------------------------------------------------------

func TestMsgServerGrantOperatorAuthorization(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	grantee2 := sdk.AccAddress([]byte("test_grantee2_______")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	futureExp := ctx.BlockTime().Add(24 * time.Hour)
	pastExp := ctx.BlockTime().Add(-1 * time.Hour)
	nowExp := ctx.BlockTime()

	testCases := []struct {
		name      string
		setupFunc func()
		msg       *types.MsgGrantOperatorAuthorization
		expectErr bool
		errContains string
	}{
		{
			name: "Valid: basic grant without operator (group proposal)",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  "",
				Grantee:   grantee,
				MsgTypes:  validMsgTypes,
			},
			expectErr: false,
		},
		{
			name: "Valid: with expiration",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority:  authority,
				Operator:   "",
				Grantee:    grantee,
				MsgTypes:   validMsgTypes,
				Expiration: &futureExp,
			},
			expectErr: false,
		},
		{
			name: "Valid: with feegrant",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority:    authority,
				Operator:     "",
				Grantee:      grantee,
				MsgTypes:     validMsgTypes,
				WithFeegrant: true,
			},
			expectErr: false,
		},
		{
			name: "Valid: with feegrant and spend limits",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority:         authority,
				Operator:          "",
				Grantee:           grantee,
				MsgTypes:          validMsgTypes,
				WithFeegrant:      true,
				FeegrantSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("stake", 500)),
			},
			expectErr: false,
		},
		{
			name: "Valid: with authz spend limit",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority:      authority,
				Operator:       "",
				Grantee:        grantee,
				MsgTypes:       validMsgTypes,
				AuthzSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
			},
			expectErr: false,
		},
		{
			name: "Invalid: expiration in the past",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority:  authority,
				Operator:   "",
				Grantee:    grantee,
				MsgTypes:   validMsgTypes,
				Expiration: &pastExp,
			},
			expectErr:   true,
			errContains: "expiration must be in the future",
		},
		{
			name: "Invalid: VSOperatorAuthorization already exists (mutual exclusivity)",
			setupFunc: func() {
				// Pre-create a VSOperatorAuthorization for authority/grantee2
				vsKey := collections.Join(authority, grantee2)
				err := k.VSOperatorAuthorizations.Set(ctx, vsKey, types.VSOperatorAuthorization{
					Authority:  authority,
					VsOperator: grantee2,
				})
				require.NoError(t, err)
			},
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  "",
				Grantee:   grantee2,
				MsgTypes:  validMsgTypes,
			},
			expectErr:   true,
			errContains: "VSOperatorAuthorization already exists",
		},
		{
			name: "Valid: with operator who has authorization",
			setupFunc: func() {
				// Grant an operator authorization so operator can execute
				operatorAddr := sdk.AccAddress([]byte("test_operator_______")).String()
				oaKey := collections.Join(authority, operatorAddr)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  operatorAddr,
					MsgTypes:  []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
				})
				require.NoError(t, err)
			},
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("test_operator_______")).String(),
				Grantee:   sdk.AccAddress([]byte("test_new_grantee____")).String(),
				MsgTypes:  validMsgTypes,
			},
			expectErr: false,
		},
		{
			name: "Invalid: operator without authorization",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("unauthorized_op_____")).String(),
				Grantee:   grantee,
				MsgTypes:  validMsgTypes,
			},
			expectErr:   true,
			errContains: "operator authorization not found",
		},
		{
			name: "Invalid: operator with expired authorization",
			setupFunc: func() {
				expiredOp := sdk.AccAddress([]byte("expired_operator____")).String()
				pastTime := ctx.BlockTime().Add(-1 * time.Hour)
				oaKey := collections.Join(authority, expiredOp)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority:  authority,
					Operator:   expiredOp,
					MsgTypes:   []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
					Expiration: &pastTime,
				})
				require.NoError(t, err)
			},
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("expired_operator____")).String(),
				Grantee:   grantee,
				MsgTypes:  validMsgTypes,
			},
			expectErr:   true,
			errContains: "operator authorization has expired",
		},
		{
			name: "Invalid: operator with wrong msg type authorization",
			setupFunc: func() {
				wrongTypeOp := sdk.AccAddress([]byte("wrong_type_operator_")).String()
				oaKey := collections.Join(authority, wrongTypeOp)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  wrongTypeOp,
					MsgTypes:  []string{"/verana.tr.v1.MsgCreateTrustRegistry"}, // wrong type
				})
				require.NoError(t, err)
			},
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("wrong_type_operator_")).String(),
				Grantee:   grantee,
				MsgTypes:  validMsgTypes,
			},
			expectErr:   true,
			errContains: "does not include requested message type",
		},
		{
			name: "Valid: multiple msg types",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority: authority,
				Operator:  "",
				Grantee:   grantee,
				MsgTypes: []string{
					"/verana.tr.v1.MsgCreateTrustRegistry",
					"/verana.cs.v1.MsgCreateCredentialSchema",
					"/verana.perm.v1.MsgCreatePermission",
				},
			},
			expectErr: false,
		},
		{
			name: "Invalid: expiration exactly at block time (boundary)",
			msg: &types.MsgGrantOperatorAuthorization{
				Authority:  authority,
				Operator:   "",
				Grantee:    grantee,
				MsgTypes:   validMsgTypes,
				Expiration: &nowExp,
			},
			expectErr:   true,
			errContains: "expiration must be in the future",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			resp, err := ms.GrantOperatorAuthorization(ctx, tc.msg)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, resp)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify OperatorAuthorization was stored
				oaKey := collections.Join(tc.msg.Authority, tc.msg.Grantee)
				oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Authority, oa.Authority)
				require.Equal(t, tc.msg.Grantee, oa.Operator) // stored as Operator field
				require.Equal(t, tc.msg.MsgTypes, oa.MsgTypes)
			}
		})
	}
}

func TestMsgServerGrantOperatorAuthorization_WithFeegrant(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// Grant with feegrant = true
	msg := &types.MsgGrantOperatorAuthorization{
		Authority:          authority,
		Grantee:            grantee,
		MsgTypes:           validMsgTypes,
		WithFeegrant:       true,
		FeegrantSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("stake", 500)),
	}
	resp, err := ms.GrantOperatorAuthorization(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify FeeGrant was created
	fgKey := collections.Join(authority, grantee)
	fg, err := k.FeeGrants.Get(ctx, fgKey)
	require.NoError(t, err)
	require.Equal(t, authority, fg.Grantor)
	require.Equal(t, grantee, fg.Grantee)
	require.Equal(t, validMsgTypes, fg.MsgTypes)

	// Now grant again with feegrant = false — should revoke existing
	msg2 := &types.MsgGrantOperatorAuthorization{
		Authority:    authority,
		Grantee:      grantee,
		MsgTypes:     validMsgTypes,
		WithFeegrant: false,
	}
	resp, err = ms.GrantOperatorAuthorization(ctx, msg2)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify FeeGrant was revoked
	has, err := k.FeeGrants.Has(ctx, fgKey)
	require.NoError(t, err)
	require.False(t, has)
}

func TestMsgServerGrantOperatorAuthorization_UpdateExisting(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	msgTypes1 := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	msgTypes2 := []string{"/verana.cs.v1.MsgCreateCredentialSchema", "/verana.perm.v1.MsgCreatePermission"}

	// Create initial
	msg1 := &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
		MsgTypes:  msgTypes1,
	}
	_, err := ms.GrantOperatorAuthorization(ctx, msg1)
	require.NoError(t, err)

	// Update with different msg types
	msg2 := &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
		MsgTypes:  msgTypes2,
	}
	_, err = ms.GrantOperatorAuthorization(ctx, msg2)
	require.NoError(t, err)

	// Verify updated
	oaKey := collections.Join(authority, grantee)
	oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
	require.NoError(t, err)
	require.Equal(t, msgTypes2, oa.MsgTypes)
}

// ---------------------------------------------------------------------------
// [MOD-DE-MSG-4] RevokeOperatorAuthorization
// ---------------------------------------------------------------------------

func TestMsgServerRevokeOperatorAuthorization(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	grantee2 := sdk.AccAddress([]byte("test_grantee2_______")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	testCases := []struct {
		name        string
		setupFunc   func()
		msg         *types.MsgRevokeOperatorAuthorization
		expectErr   bool
		errContains string
	}{
		{
			name: "Valid: revoke existing authorization (group proposal, no operator)",
			setupFunc: func() {
				// Create OperatorAuthorization to revoke
				oaKey := collections.Join(authority, grantee)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  grantee,
					MsgTypes:  validMsgTypes,
				})
				require.NoError(t, err)
			},
			msg: &types.MsgRevokeOperatorAuthorization{
				Authority: authority,
				Operator:  "",
				Grantee:   grantee,
			},
			expectErr: false,
		},
		{
			name: "Invalid: authorization does not exist",
			msg: &types.MsgRevokeOperatorAuthorization{
				Authority: authority,
				Operator:  "",
				Grantee:   sdk.AccAddress([]byte("nonexistent_grantee_")).String(),
			},
			expectErr:   true,
			errContains: "operator authorization not found for this authority/grantee pair",
		},
		{
			name: "Invalid: operator without authorization",
			setupFunc: func() {
				// Create the OA that we want to revoke
				oaKey := collections.Join(authority, grantee2)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  grantee2,
					MsgTypes:  validMsgTypes,
				})
				require.NoError(t, err)
			},
			msg: &types.MsgRevokeOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("unauthorized_op_____")).String(),
				Grantee:   grantee2,
			},
			expectErr:   true,
			errContains: "operator authorization not found",
		},
		{
			name: "Valid: operator with proper authorization",
			setupFunc: func() {
				// Create OA for operator to revoke with
				operatorAddr := sdk.AccAddress([]byte("revoking_operator___")).String()
				opKey := collections.Join(authority, operatorAddr)
				err := k.OperatorAuthorizations.Set(ctx, opKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  operatorAddr,
					MsgTypes:  []string{"/verana.de.v1.MsgRevokeOperatorAuthorization"},
				})
				require.NoError(t, err)

				// Create OA to be revoked
				targetGrantee := sdk.AccAddress([]byte("target_grantee______")).String()
				targetKey := collections.Join(authority, targetGrantee)
				err = k.OperatorAuthorizations.Set(ctx, targetKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  targetGrantee,
					MsgTypes:  validMsgTypes,
				})
				require.NoError(t, err)
			},
			msg: &types.MsgRevokeOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("revoking_operator___")).String(),
				Grantee:   sdk.AccAddress([]byte("target_grantee______")).String(),
			},
			expectErr: false,
		},
		{
			name: "Invalid: operator with expired authorization",
			setupFunc: func() {
				expiredOp := sdk.AccAddress([]byte("expired_revoke_op___")).String()
				pastTime := ctx.BlockTime().Add(-1 * time.Hour)
				opKey := collections.Join(authority, expiredOp)
				err := k.OperatorAuthorizations.Set(ctx, opKey, types.OperatorAuthorization{
					Authority:  authority,
					Operator:   expiredOp,
					MsgTypes:   []string{"/verana.de.v1.MsgRevokeOperatorAuthorization"},
					Expiration: &pastTime,
				})
				require.NoError(t, err)

				// Create OA to be revoked
				revokeTarget := sdk.AccAddress([]byte("revoke_target_______")).String()
				targetKey := collections.Join(authority, revokeTarget)
				err = k.OperatorAuthorizations.Set(ctx, targetKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  revokeTarget,
					MsgTypes:  validMsgTypes,
				})
				require.NoError(t, err)
			},
			msg: &types.MsgRevokeOperatorAuthorization{
				Authority: authority,
				Operator:  sdk.AccAddress([]byte("expired_revoke_op___")).String(),
				Grantee:   sdk.AccAddress([]byte("revoke_target_______")).String(),
			},
			expectErr:   true,
			errContains: "operator authorization has expired",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			resp, err := ms.RevokeOperatorAuthorization(ctx, tc.msg)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, resp)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify OperatorAuthorization was removed
				oaKey := collections.Join(tc.msg.Authority, tc.msg.Grantee)
				has, err := k.OperatorAuthorizations.Has(ctx, oaKey)
				require.NoError(t, err)
				require.False(t, has)
			}
		})
	}
}

func TestMsgServerRevokeOperatorAuthorization_AlsoRevokesFeeGrant(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// First, grant operator authorization with feegrant
	grantMsg := &types.MsgGrantOperatorAuthorization{
		Authority:    authority,
		Grantee:      grantee,
		MsgTypes:     validMsgTypes,
		WithFeegrant: true,
	}
	_, err := ms.GrantOperatorAuthorization(ctx, grantMsg)
	require.NoError(t, err)

	// Verify both OA and FeeGrant exist
	oaKey := collections.Join(authority, grantee)
	has, err := k.OperatorAuthorizations.Has(ctx, oaKey)
	require.NoError(t, err)
	require.True(t, has)

	fgKey := collections.Join(authority, grantee)
	has, err = k.FeeGrants.Has(ctx, fgKey)
	require.NoError(t, err)
	require.True(t, has)

	// Revoke operator authorization
	revokeMsg := &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
	}
	_, err = ms.RevokeOperatorAuthorization(ctx, revokeMsg)
	require.NoError(t, err)

	// Verify both OA and FeeGrant were removed
	has, err = k.OperatorAuthorizations.Has(ctx, oaKey)
	require.NoError(t, err)
	require.False(t, has)

	has, err = k.FeeGrants.Has(ctx, fgKey)
	require.NoError(t, err)
	require.False(t, has)
}

func TestMsgServerRevokeOperatorAuthorization_NoFeeGrant(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// Grant operator authorization without feegrant
	grantMsg := &types.MsgGrantOperatorAuthorization{
		Authority:    authority,
		Grantee:      grantee,
		MsgTypes:     validMsgTypes,
		WithFeegrant: false,
	}
	_, err := ms.GrantOperatorAuthorization(ctx, grantMsg)
	require.NoError(t, err)

	// Verify OA exists but FeeGrant doesn't
	oaKey := collections.Join(authority, grantee)
	has, err := k.OperatorAuthorizations.Has(ctx, oaKey)
	require.NoError(t, err)
	require.True(t, has)

	fgKey := collections.Join(authority, grantee)
	has, err = k.FeeGrants.Has(ctx, fgKey)
	require.NoError(t, err)
	require.False(t, has)

	// Revoke should succeed even without FeeGrant
	revokeMsg := &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
	}
	_, err = ms.RevokeOperatorAuthorization(ctx, revokeMsg)
	require.NoError(t, err)

	// Verify OA was removed
	has, err = k.OperatorAuthorizations.Has(ctx, oaKey)
	require.NoError(t, err)
	require.False(t, has)
}

// ---------------------------------------------------------------------------
// [AUTHZ-CHECK-1] CheckOperatorAuthorization
// ---------------------------------------------------------------------------

func TestCheckOperatorAuthorization(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	now := ctx.BlockTime()
	futureExp := now.Add(24 * time.Hour)
	pastExp := now.Add(-1 * time.Hour)

	testCases := []struct {
		name        string
		setupFunc   func()
		authority   string
		operator    string
		msgTypeURL  string
		expectErr   bool
		errContains string
	}{
		{
			name:       "Valid: empty operator skips check (group proposal)",
			authority:  authority,
			operator:   "",
			msgTypeURL: "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:  false,
		},
		{
			name: "Valid: operator with matching authorization",
			setupFunc: func() {
				oaKey := collections.Join(authority, operator)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  operator,
					MsgTypes:  []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
				})
				require.NoError(t, err)
			},
			authority:  authority,
			operator:   operator,
			msgTypeURL: "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:  false,
		},
		{
			name: "Valid: operator with future expiration",
			setupFunc: func() {
				op2 := sdk.AccAddress([]byte("op_future_exp_______")).String()
				oaKey := collections.Join(authority, op2)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority:  authority,
					Operator:   op2,
					MsgTypes:   []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
					Expiration: &futureExp,
				})
				require.NoError(t, err)
			},
			authority:  authority,
			operator:   sdk.AccAddress([]byte("op_future_exp_______")).String(),
			msgTypeURL: "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:  false,
		},
		{
			name:        "Invalid: no authorization exists",
			authority:   authority,
			operator:    sdk.AccAddress([]byte("no_authz_operator___")).String(),
			msgTypeURL:  "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:   true,
			errContains: "operator authorization not found",
		},
		{
			name: "Invalid: authorization expired",
			setupFunc: func() {
				expOp := sdk.AccAddress([]byte("authz_expired_op____")).String()
				oaKey := collections.Join(authority, expOp)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority:  authority,
					Operator:   expOp,
					MsgTypes:   []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
					Expiration: &pastExp,
				})
				require.NoError(t, err)
			},
			authority:   authority,
			operator:    sdk.AccAddress([]byte("authz_expired_op____")).String(),
			msgTypeURL:  "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:   true,
			errContains: "operator authorization has expired",
		},
		{
			name: "Invalid: msg type not in authorization",
			setupFunc: func() {
				wrongOp := sdk.AccAddress([]byte("wrong_msgtype_op____")).String()
				oaKey := collections.Join(authority, wrongOp)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  wrongOp,
					MsgTypes:  []string{"/verana.tr.v1.MsgCreateTrustRegistry"},
				})
				require.NoError(t, err)
			},
			authority:   authority,
			operator:    sdk.AccAddress([]byte("wrong_msgtype_op____")).String(),
			msgTypeURL:  "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:   true,
			errContains: "does not include requested message type",
		},
		{
			name: "Valid: operator with multiple msg types including requested",
			setupFunc: func() {
				multiOp := sdk.AccAddress([]byte("multi_msgtype_op____")).String()
				oaKey := collections.Join(authority, multiOp)
				err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
					Authority: authority,
					Operator:  multiOp,
					MsgTypes: []string{
						"/verana.tr.v1.MsgCreateTrustRegistry",
						"/verana.de.v1.MsgGrantOperatorAuthorization",
						"/verana.cs.v1.MsgCreateCredentialSchema",
					},
				})
				require.NoError(t, err)
			},
			authority:  authority,
			operator:   sdk.AccAddress([]byte("multi_msgtype_op____")).String(),
			msgTypeURL: "/verana.de.v1.MsgGrantOperatorAuthorization",
			expectErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			err := k.CheckOperatorAuthorization(ctx, tc.authority, tc.operator, tc.msgTypeURL, now)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration / end-to-end scenarios
// ---------------------------------------------------------------------------

func TestGrantThenRevokeOperatorAuthorization_E2E(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{
		"/verana.tr.v1.MsgCreateTrustRegistry",
		"/verana.cs.v1.MsgCreateCredentialSchema",
	}

	// Step 1: Grant with feegrant
	grantMsg := &types.MsgGrantOperatorAuthorization{
		Authority:          authority,
		Grantee:            grantee,
		MsgTypes:           validMsgTypes,
		WithFeegrant:       true,
		FeegrantSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
	}
	resp, err := ms.GrantOperatorAuthorization(ctx, grantMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify both exist
	oaKey := collections.Join(authority, grantee)
	oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
	require.NoError(t, err)
	require.Equal(t, validMsgTypes, oa.MsgTypes)

	fgKey := collections.Join(authority, grantee)
	fg, err := k.FeeGrants.Get(ctx, fgKey)
	require.NoError(t, err)
	require.Equal(t, authority, fg.Grantor)

	// Step 2: Revoke
	revokeResp, err := ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
	})
	require.NoError(t, err)
	require.NotNil(t, revokeResp)

	// Verify both removed
	has, err := k.OperatorAuthorizations.Has(ctx, oaKey)
	require.NoError(t, err)
	require.False(t, has)

	has, err = k.FeeGrants.Has(ctx, fgKey)
	require.NoError(t, err)
	require.False(t, has)

	// Step 3: Revoke again should fail (no authorization exists)
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator authorization not found for this authority/grantee pair")
}

func TestMutualExclusivity_OAAndVSOA(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// First, create a VSOperatorAuthorization
	vsKey := collections.Join(authority, grantee)
	err := k.VSOperatorAuthorizations.Set(ctx, vsKey, types.VSOperatorAuthorization{
		Authority:  authority,
		VsOperator: grantee,
	})
	require.NoError(t, err)

	// Attempt to grant OperatorAuthorization should fail
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
		MsgTypes:  validMsgTypes,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "VSOperatorAuthorization already exists")

	// Remove VSOperatorAuthorization
	err = k.VSOperatorAuthorizations.Remove(ctx, vsKey)
	require.NoError(t, err)

	// Now grant should succeed
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
		MsgTypes:  validMsgTypes,
	})
	require.NoError(t, err)
}

func TestMultipleGranteesForSameAuthority(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee1 := sdk.AccAddress([]byte("test_grantee1_______")).String()
	grantee2 := sdk.AccAddress([]byte("test_grantee2_______")).String()
	grantee3 := sdk.AccAddress([]byte("test_grantee3_______")).String()
	msgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// Grant to multiple grantees
	for _, grantee := range []string{grantee1, grantee2, grantee3} {
		_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
			Authority: authority,
			Grantee:   grantee,
			MsgTypes:  msgTypes,
		})
		require.NoError(t, err)
	}

	// Verify all exist
	for _, grantee := range []string{grantee1, grantee2, grantee3} {
		oaKey := collections.Join(authority, grantee)
		has, err := k.OperatorAuthorizations.Has(ctx, oaKey)
		require.NoError(t, err)
		require.True(t, has)
	}

	// Revoke one
	_, err := ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee2,
	})
	require.NoError(t, err)

	// Verify grantee2 removed, others still exist
	has, err := k.OperatorAuthorizations.Has(ctx, collections.Join(authority, grantee2))
	require.NoError(t, err)
	require.False(t, has)

	for _, grantee := range []string{grantee1, grantee3} {
		oaKey := collections.Join(authority, grantee)
		has, err := k.OperatorAuthorizations.Has(ctx, oaKey)
		require.NoError(t, err)
		require.True(t, has)
	}
}

func TestGrantRevokeReGrant_E2E(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	msgTypes1 := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	msgTypes2 := []string{"/verana.cs.v1.MsgCreateCredentialSchema"}

	// Grant
	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
		MsgTypes:  msgTypes1,
	})
	require.NoError(t, err)

	// Revoke
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
	})
	require.NoError(t, err)

	// Re-grant with different msg types
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   grantee,
		MsgTypes:  msgTypes2,
	})
	require.NoError(t, err)

	// Verify new grant has correct msg types
	oaKey := collections.Join(authority, grantee)
	oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
	require.NoError(t, err)
	require.Equal(t, msgTypes2, oa.MsgTypes)
}

func TestAuthorityIsolation(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority1 := sdk.AccAddress([]byte("test_authority1_____")).String()
	authority2 := sdk.AccAddress([]byte("test_authority2_____")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	msgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}

	// Grant from authority1
	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority1,
		Grantee:   grantee,
		MsgTypes:  msgTypes,
	})
	require.NoError(t, err)

	// Grant from authority2 (same grantee, different authority)
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority2,
		Grantee:   grantee,
		MsgTypes:  msgTypes,
	})
	require.NoError(t, err)

	// Verify both exist independently
	has1, err := k.OperatorAuthorizations.Has(ctx, collections.Join(authority1, grantee))
	require.NoError(t, err)
	require.True(t, has1)

	has2, err := k.OperatorAuthorizations.Has(ctx, collections.Join(authority2, grantee))
	require.NoError(t, err)
	require.True(t, has2)

	// Revoke from authority1 should NOT affect authority2
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority1,
		Grantee:   grantee,
	})
	require.NoError(t, err)

	has1, err = k.OperatorAuthorizations.Has(ctx, collections.Join(authority1, grantee))
	require.NoError(t, err)
	require.False(t, has1)

	has2, err = k.OperatorAuthorizations.Has(ctx, collections.Join(authority2, grantee))
	require.NoError(t, err)
	require.True(t, has2)

	// authority1 cannot revoke authority2's grant
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority1,
		Grantee:   grantee,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator authorization not found for this authority/grantee pair")
}

func TestOperatorRevokesOwnAuthorization(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()
	msgTypes := []string{
		"/verana.tr.v1.MsgCreateTrustRegistry",
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
	}

	// Grant operator authorization (includes revoke permission)
	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Grantee:   operator,
		MsgTypes:  msgTypes,
	})
	require.NoError(t, err)

	// Operator revokes their own authorization
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: authority,
		Operator:  operator,
		Grantee:   operator,
	})
	require.NoError(t, err)

	// Verify removed
	has, err := k.OperatorAuthorizations.Has(ctx, collections.Join(authority, operator))
	require.NoError(t, err)
	require.False(t, has)
}

func TestCheckOperatorAuthorization_ExpirationBoundary(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("boundary_op_________")).String()
	now := ctx.BlockTime()

	// Expiration exactly at block time — should fail (!After(now) is true when equal)
	exactNow := now
	oaKey := collections.Join(authority, operator)
	err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
		Authority:  authority,
		Operator:   operator,
		MsgTypes:   []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
		Expiration: &exactNow,
	})
	require.NoError(t, err)

	err = k.CheckOperatorAuthorization(ctx, authority, operator,
		"/verana.de.v1.MsgGrantOperatorAuthorization", now)
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator authorization has expired")

	// Expiration 1 nanosecond after block time — should pass
	oneNsAfter := now.Add(1 * time.Nanosecond)
	err = k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
		Authority:  authority,
		Operator:   operator,
		MsgTypes:   []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
		Expiration: &oneNsAfter,
	})
	require.NoError(t, err)

	err = k.CheckOperatorAuthorization(ctx, authority, operator,
		"/verana.de.v1.MsgGrantOperatorAuthorization", now)
	require.NoError(t, err)
}

func TestRevokeFeeAllowance_DoubleRevoke(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()

	// Create a fee grant
	err := k.GrantFeeAllowance(ctx, authority, grantee,
		[]string{"/verana.tr.v1.MsgCreateTrustRegistry"}, nil, nil, nil)
	require.NoError(t, err)

	// First revoke
	err = k.RevokeFeeAllowance(ctx, authority, grantee)
	require.NoError(t, err)

	// Verify removed
	has, err := k.FeeGrants.Has(ctx, collections.Join(authority, grantee))
	require.NoError(t, err)
	require.False(t, has)

	// Second revoke should be no-op (not an error)
	err = k.RevokeFeeAllowance(ctx, authority, grantee)
	require.NoError(t, err)
}

func TestGrantOperatorAuthorization_FeegrantFieldsStoredCorrectly(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	grantee := sdk.AccAddress([]byte("test_grantee________")).String()
	validMsgTypes := []string{"/verana.tr.v1.MsgCreateTrustRegistry"}
	futureExp := ctx.BlockTime().Add(24 * time.Hour)
	spendLimit := sdk.NewCoins(sdk.NewInt64Coin("stake", 500))
	period := 12 * time.Hour

	// Grant with all feegrant fields
	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority:                authority,
		Grantee:                  grantee,
		MsgTypes:                 validMsgTypes,
		Expiration:               &futureExp,
		AuthzSpendLimit:          sdk.NewCoins(sdk.NewInt64Coin("stake", 1000)),
		WithFeegrant:             true,
		FeegrantSpendLimit:       spendLimit,
		FeegrantSpendLimitPeriod: &period,
	})
	require.NoError(t, err)

	// Verify OA stored correctly
	oaKey := collections.Join(authority, grantee)
	oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
	require.NoError(t, err)
	require.Equal(t, validMsgTypes, oa.MsgTypes)
	require.True(t, oa.SpendLimit.Equal(sdk.NewCoins(sdk.NewInt64Coin("stake", 1000))))
	require.NotNil(t, oa.Expiration)
	require.True(t, oa.Expiration.Equal(futureExp))

	// Verify FeeGrant stored correctly
	fgKey := collections.Join(authority, grantee)
	fg, err := k.FeeGrants.Get(ctx, fgKey)
	require.NoError(t, err)
	require.Equal(t, authority, fg.Grantor)
	require.Equal(t, grantee, fg.Grantee)
	require.Equal(t, validMsgTypes, fg.MsgTypes)
	require.True(t, fg.SpendLimit.Equal(spendLimit))
	require.NotNil(t, fg.Period)
	require.Equal(t, period, *fg.Period)
}

// ---------------------------------------------------------------------------
// Spec concern: Privilege escalation via operator self-grant
// ---------------------------------------------------------------------------

// TestOperatorPrivilegeEscalation_SelfGrant demonstrates that an operator
// who only has MsgGrantOperatorAuthorization permission can escalate their
// own privileges by granting themselves additional msg_types.
//
// This is a potential spec concern: AUTHZ-CHECK only verifies the operator
// has permission for the current message type (MsgGrantOperatorAuthorization),
// not that the granted msg_types are a subset of the operator's own permissions.
//
// WARNING: This test documents the current behavior. If the spec considers
// this undesirable, a check should be added to verify that an operator cannot
// grant msg_types beyond their own authorization scope.
func TestOperatorPrivilegeEscalation_SelfGrant(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := sdk.AccAddress([]byte("test_operator_______")).String()

	// Operator has ONLY MsgGrantOperatorAuthorization permission
	oaKey := collections.Join(authority, operator)
	err := k.OperatorAuthorizations.Set(ctx, oaKey, types.OperatorAuthorization{
		Authority: authority,
		Operator:  operator,
		MsgTypes:  []string{"/verana.de.v1.MsgGrantOperatorAuthorization"},
	})
	require.NoError(t, err)

	// Operator grants THEMSELVES all VPR delegable msg types
	allMsgTypes := []string{
		"/verana.tr.v1.MsgCreateTrustRegistry",
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.perm.v1.MsgCreatePermission",
		"/verana.de.v1.MsgGrantOperatorAuthorization",
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
	}

	// This SUCCEEDS — the operator overwrites their own authorization
	resp, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: authority,
		Operator:  operator,
		Grantee:   operator, // grantee == operator (self-grant)
		MsgTypes:  allMsgTypes,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify the operator now has escalated privileges
	oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
	require.NoError(t, err)
	require.Equal(t, allMsgTypes, oa.MsgTypes) // escalated from 1 to 5 msg types

	// The operator can now execute msg types they originally didn't have
	err = k.CheckOperatorAuthorization(ctx, authority, operator,
		"/verana.tr.v1.MsgCreateTrustRegistry", ctx.BlockTime())
	require.NoError(t, err) // passes — privilege escalated
}

// ---------------------------------------------------------------------------
// Bootstrap flow: group proposal → first operator → subsequent operators
// ---------------------------------------------------------------------------

// TestBootstrapFlow_GroupProposalOnboardsFirstOperator models the real-world
// onboarding sequence per the VPR spec:
//
//  1. A group proposal (operator="") onboards the first operator.
//  2. That operator then onboards a second operator on behalf of the authority.
//  3. The second operator can execute delegated messages.
//  4. The group can also directly revoke any operator (operator="").
func TestBootstrapFlow_GroupProposalOnboardsFirstOperator(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	// "groupAccount" represents the x/group policy address.
	// In production this is generated by the group module; in unit tests it's
	// just an AccAddress — the DE module never calls the group keeper.
	groupAccount := sdk.AccAddress([]byte("group_policy_addr___")).String()

	firstOperator := sdk.AccAddress([]byte("first_operator______")).String()
	secondOperator := sdk.AccAddress([]byte("second_operator_____")).String()

	deMsgTypes := []string{
		"/verana.de.v1.MsgGrantOperatorAuthorization",
		"/verana.de.v1.MsgRevokeOperatorAuthorization",
	}
	trMsgTypes := []string{
		"/verana.tr.v1.MsgCreateTrustRegistry",
	}

	// ---- Step 1: Group proposal onboards first operator ----
	// Operator field is empty → simulates group governance execution.
	_, err := ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority:    groupAccount,
		Operator:     "", // group proposal — no operator cosigner
		Grantee:      firstOperator,
		MsgTypes:     append(deMsgTypes, trMsgTypes...),
		WithFeegrant: true,
	})
	require.NoError(t, err)

	// Verify first operator was onboarded
	oaKey := collections.Join(groupAccount, firstOperator)
	oa, err := k.OperatorAuthorizations.Get(ctx, oaKey)
	require.NoError(t, err)
	require.Equal(t, groupAccount, oa.Authority)
	require.Equal(t, firstOperator, oa.Operator)

	// ---- Step 2: First operator onboards second operator ----
	// First operator acts on behalf of the group account (operator field set).
	_, err = ms.GrantOperatorAuthorization(ctx, &types.MsgGrantOperatorAuthorization{
		Authority: groupAccount,
		Operator:  firstOperator, // existing operator cosigns
		Grantee:   secondOperator,
		MsgTypes:  trMsgTypes, // only TR permissions
	})
	require.NoError(t, err)

	// Verify second operator was onboarded with limited permissions
	oaKey2 := collections.Join(groupAccount, secondOperator)
	oa2, err := k.OperatorAuthorizations.Get(ctx, oaKey2)
	require.NoError(t, err)
	require.Equal(t, trMsgTypes, oa2.MsgTypes)

	// ---- Step 3: Second operator can use delegated msg type ----
	err = k.CheckOperatorAuthorization(ctx, groupAccount, secondOperator,
		"/verana.tr.v1.MsgCreateTrustRegistry", ctx.BlockTime())
	require.NoError(t, err)

	// But second operator CANNOT grant further operators (not in their msg_types)
	err = k.CheckOperatorAuthorization(ctx, groupAccount, secondOperator,
		"/verana.de.v1.MsgGrantOperatorAuthorization", ctx.BlockTime())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not include requested message type")

	// ---- Step 4: Group can directly revoke any operator (group proposal) ----
	_, err = ms.RevokeOperatorAuthorization(ctx, &types.MsgRevokeOperatorAuthorization{
		Authority: groupAccount,
		Operator:  "", // group proposal — direct revocation
		Grantee:   secondOperator,
	})
	require.NoError(t, err)

	// Verify second operator was removed
	has, err := k.OperatorAuthorizations.Has(ctx, oaKey2)
	require.NoError(t, err)
	require.False(t, has)

	// First operator still exists
	has, err = k.OperatorAuthorizations.Has(ctx, oaKey)
	require.NoError(t, err)
	require.True(t, has)
}
