package keeper_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cstypes "github.com/verana-labs/verana/x/cs/types"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/pp/keeper"
	"github.com/verana-labs/verana/x/pp/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockCredentialSchemaKeeper, *keepertest.MockPermEcosystemKeeper, context.Context) {
	k, csKeeper, trkKeeper, _, ctx, _ := keepertest.PermissionKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx
}

func setupMsgServerWithDelegation(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockCredentialSchemaKeeper, *keepertest.MockPermEcosystemKeeper, context.Context, *keepertest.MockDelegationKeeper) {
	k, csKeeper, trkKeeper, _, ctx, delKeeper := keepertest.PermissionKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx, delKeeper
}

func TestMsgServer(t *testing.T) {
	k, ms, _, _, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

// Test for StartParticipantOP
func TestStartPermissionVP(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	creator2 := sdk.AccAddress([]byte("test_creator_two")).String()
	creator3 := sdk.AccAddress([]byte("test_creator_thr")).String()
	creator4 := sdk.AccAddress([]byte("test_creator_fou")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry for our credential schema
	trID := trkKeeper.CreateMockEcosystem(creator, validDid)

	// Create mock credential schema with specific perm management modes
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	// Create validator perm (ISSUER_GRANTOR)
	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE
	// This should be VALIDATED as it's a prerequisite
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED, // validator must be validated
		EffectiveFrom: &pastTime,                       // Required for ACTIVE state
	}

	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create another validator perm (VERIFIER_GRANTOR with different country)
	verifierGrantorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_VERIFIER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	verifierGrantorPermID, err := k.CreatePermission(sdkCtx, verifierGrantorPerm)
	require.NoError(t, err)

	// Create a validator perm without country (for testing optional country)
	validatorPermNoCountry := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	validatorPermNoCountryID, err := k.CreatePermission(sdkCtx, validatorPermNoCountry)
	require.NoError(t, err)

	testCases := []struct {
		name                     string
		msg                      *types.MsgStartParticipantOP
		err                      string
		checkFees                bool
		expectedValidationFees   uint64
		expectedIssuanceFees     uint64
		expectedVerificationFees uint64
	}{
		{
			name: "Valid ISSUER Participant Request",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: validatorPermID,
				Did:                    validDid,
			},
			err:       "",
			checkFees: false,
		},
		{
			name: "Valid ISSUER Participant Request with optional fees",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator2,
				Operator:               creator2,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: validatorPermID,
				Did:                    validDid,
				ValidationFees:         &types.OptionalUInt64{Value: 100},
				IssuanceFees:           &types.OptionalUInt64{Value: 50},
				VerificationFees:       &types.OptionalUInt64{Value: 25},
			},
			err:                      "",
			checkFees:                true,
			expectedValidationFees:   100,
			expectedIssuanceFees:     50,
			expectedVerificationFees: 25,
		},
		{
			name: "Valid ISSUER Participant Request with partial fees",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator3,
				Operator:               creator3,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: validatorPermID,
				Did:                    validDid,
				ValidationFees:         &types.OptionalUInt64{Value: 75},
			},
			err:                      "",
			checkFees:                true,
			expectedValidationFees:   75,
			expectedIssuanceFees:     0,
			expectedVerificationFees: 0,
		},
		{
			name: "Valid ISSUER Participant Request with zero fees",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator4,
				Operator:               creator4,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: validatorPermID,
				Did:                    validDid,
				ValidationFees:         &types.OptionalUInt64{Value: 0},
				IssuanceFees:           &types.OptionalUInt64{Value: 0},
				VerificationFees:       &types.OptionalUInt64{Value: 0},
			},
			err:                      "",
			checkFees:                true,
			expectedValidationFees:   0,
			expectedIssuanceFees:     0,
			expectedVerificationFees: 0,
		},
		{
			name: "Valid ISSUER Participant Request without country on validator",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: validatorPermNoCountryID,
				Did:                    validDid,
			},
			err:       "",
			checkFees: false,
		},
		{
			name: "Non-existent Validator Participant",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: 999,
				Did:                    validDid,
			},
			err:       "validator perm not found",
			checkFees: false,
		},
		{
			name: "Invalid Participant Type Combination - ISSUER with wrong validator",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: verifierGrantorPermID, // Wrong validator type
				Did:                    validDid,
			},
			err:       "issuer perm requires ISSUER_GRANTOR validator",
			checkFees: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.StartParticipantOP(ctx, tc.msg)
			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Greater(t, resp.ParticipantId, uint64(0))

				// Verify created perm
				perm, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Role, perm.Role)
				require.NotZero(t, perm.CorporationId)
				require.Equal(t, tc.msg.ValidatorParticipantId, perm.ValidatorParticipantId)
				require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
				require.NotNil(t, perm.Created)
				require.NotNil(t, perm.Modified)
				require.NotNil(t, perm.OpLastStateChange)

				// Verify requested fees if provided
				if tc.checkFees {
					require.Equal(t, tc.expectedValidationFees, perm.ValidationFees, "Validation fees should match requested value")
					require.Equal(t, tc.expectedIssuanceFees, perm.IssuanceFees, "Issuance fees should match requested value")
					require.Equal(t, tc.expectedVerificationFees, perm.VerificationFees, "Verification fees should match requested value")
				} else {
					// If fees were not provided, they should be 0
					require.Equal(t, uint64(0), perm.ValidationFees, "Validation fees should be 0 when not provided")
					require.Equal(t, uint64(0), perm.IssuanceFees, "Issuance fees should be 0 when not provided")
					require.Equal(t, uint64(0), perm.VerificationFees, "Verification fees should be 0 when not provided")
				}
			}
		})
	}
}

func TestRenewPermissionVP(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	// Create validator perm
	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          3, // ISSUER_GRANTOR
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}

	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)

	require.NoError(t, err)

	// Create applicant perm
	applicantPerm := types.Participant{
		SchemaId:               1,
		Role:                   1, // ISSUER
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdk.UnwrapSDKContext(ctx), applicantPerm)
	require.NoError(t, err)

	testCases := []struct {
		name string
		msg  *types.MsgRenewParticipantOP
		err  string
	}{
		{
			name: "Non-existent Participant",
			msg: &types.MsgRenewParticipantOP{
				Corporation: creator,
				Operator:    creator,
				Id:          999,
			},
			err: "perm not found",
		},
		{
			name: "Wrong Authority",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("wrong_creator")).String(),
				Operator:    sdk.AccAddress([]byte("wrong_creator")).String(),
				Id:          applicantPermID,
			},
			err: "authority is not the participant authority",
		},
		{
			name: "Successful Renewal",
			msg: &types.MsgRenewParticipantOP{
				Corporation: creator,
				Operator:    creator,
				Id:          applicantPermID,
			},
			err: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RenewParticipantOP(ctx, tc.msg)
			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify updated perm
				perm, err := k.GetParticipantByID(sdk.UnwrapSDKContext(ctx), tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
				require.NotNil(t, perm.OpLastStateChange)
			}
		})
	}
}

func TestRenewPermissionVP_AuthzCheck(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, mockDelegation := setupMsgServerWithDelegation(t)
	_ = trkKeeper

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          3,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	applicantPerm := types.Participant{
		SchemaId:               1,
		Role:                   1,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	t.Run("AUTHZ-CHECK failure blocks renewal", func(t *testing.T) {
		mockDelegation.ErrToReturn = fmt.Errorf("operator not authorized")
		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Nil(t, resp)
	})

	t.Run("AUTHZ-CHECK success allows renewal", func(t *testing.T) {
		mockDelegation.ErrToReturn = nil
		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
	})
}

func TestRenewPermissionVP_VpStatePrecondition(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          3,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("Renewing PENDING perm is blocked (prevents fee accounting loss)", func(t *testing.T) {
		pendingPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          1000, // funds already in escrow
			OpCurrentDeposit:       500,
		}
		pendingPermID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          pendingPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "op_state must be VALIDATED to renew")
		require.Nil(t, resp)

		// Verify the perm was NOT modified (fees still intact)
		perm, err := k.GetParticipantByID(sdkCtx, pendingPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
		require.Equal(t, uint64(1000), perm.OpCurrentFees)
		require.Equal(t, uint64(500), perm.OpCurrentDeposit)
	})

	t.Run("Renewing UNSPECIFIED op_state perm is blocked", func(t *testing.T) {
		unspecPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_ONBOARDING_STATE_UNSPECIFIED,
		}
		unspecPermID, err := k.CreatePermission(sdkCtx, unspecPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          unspecPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "op_state must be VALIDATED to renew")
		require.Nil(t, resp)
	})

	t.Run("Renewing VALIDATED perm succeeds", func(t *testing.T) {
		validatedPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		validatedPermID, err := k.CreatePermission(sdkCtx, validatedPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          validatedPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, validatedPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
		require.NotNil(t, perm.OpLastStateChange)
		require.Equal(t, now, *perm.OpLastStateChange)
		require.Equal(t, now, *perm.Modified)
	})
}

func TestRenewPermissionVP_ValidatorPermChecks(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	t.Run("Renewal blocked when validator perm is revoked", func(t *testing.T) {
		revokedTime := now.Add(-30 * time.Minute)
		revokedValidatorPerm := types.Participant{
			SchemaId:      1,
			Role:          3,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
			Revoked:       &revokedTime,
		}
		revokedValPermID, err := k.CreatePermission(sdkCtx, revokedValidatorPerm)
		require.NoError(t, err)

		applicantPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: revokedValPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator perm is expired", func(t *testing.T) {
		expiredTime := now.Add(-10 * time.Minute)
		expiredValidatorPerm := types.Participant{
			SchemaId:       1,
			Role:           3,
			CorporationId:  trkKeeper.RegisterCorp(creator),
			Created:        &now,
			Modified:       &now,
			OpState:        types.OnboardingState_VALIDATED,
			EffectiveFrom:  &pastTime,
			EffectiveUntil: &expiredTime,
		}
		expiredValPermID, err := k.CreatePermission(sdkCtx, expiredValidatorPerm)
		require.NoError(t, err)

		applicantPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: expiredValPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator perm is INACTIVE (no effective_from)", func(t *testing.T) {
		inactiveValidatorPerm := types.Participant{
			SchemaId:      1,
			Role:          3,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			// EffectiveFrom is nil => INACTIVE
		}
		inactiveValPermID, err := k.CreatePermission(sdkCtx, inactiveValidatorPerm)
		require.NoError(t, err)

		applicantPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: inactiveValPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator perm does not exist", func(t *testing.T) {
		applicantPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: 99999, // non-existent
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm not found")
		require.Nil(t, resp)
	})
}

func TestRenewPermissionVP_FeeAndDepositAccumulation(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	// MockTrustRegistryKeeper returns trust_unit_price=1 by default
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorPerm := types.Participant{
		SchemaId:       1,
		Role:           3,
		CorporationId:  trkKeeper.RegisterCorp(creator),
		Created:        &now,
		Adjusted:       &now,
		Modified:       &now,
		OpState:        types.OnboardingState_VALIDATED,
		EffectiveFrom:  &pastTime,
		ValidationFees: 50, // 50 trust units * 1 price = 50 denom fees
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("Deposit accumulates on renewal", func(t *testing.T) {
		initialDeposit := uint64(100)
		applicantPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			Deposit:                initialDeposit,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
		// Deposit should accumulate: initialDeposit + new deposit
		require.True(t, perm.Deposit >= initialDeposit, "deposit should accumulate, got %d", perm.Deposit)
		require.True(t, perm.OpCurrentFees > 0 || perm.OpCurrentDeposit > 0 || validatorPerm.ValidationFees == 0,
			"current fees or deposit should be set based on validator fees")
	})

	t.Run("Different operator than authority is allowed", func(t *testing.T) {
		operator := sdk.AccAddress([]byte("different_oper")).String()
		applicantPerm := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    operator,
			Id:          applicantPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
	})
}

func TestRenewPermissionVP_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name string
		msg  *types.MsgRenewParticipantOP
		err  string
	}{
		{
			name: "Empty authority address",
			msg: &types.MsgRenewParticipantOP{
				Corporation: "",
				Operator:    sdk.AccAddress([]byte("test_operator")).String(),
				Id:          1,
			},
			err: "invalid corporation address",
		},
		{
			name: "Invalid authority address",
			msg: &types.MsgRenewParticipantOP{
				Corporation: "invalid_address",
				Operator:    sdk.AccAddress([]byte("test_operator")).String(),
				Id:          1,
			},
			err: "invalid corporation address",
		},
		{
			name: "Empty operator address",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:    "",
				Id:          1,
			},
			err: "invalid operator address",
		},
		{
			name: "Invalid operator address",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:    "invalid_address",
				Id:          1,
			},
			err: "invalid operator address",
		},
		{
			name: "Zero perm ID",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:    sdk.AccAddress([]byte("test_operator")).String(),
				Id:          0,
			},
			err: "perm ID cannot be 0",
		},
		{
			name: "Valid message",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:    sdk.AccAddress([]byte("test_operator")).String(),
				Id:          1,
			},
			err: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSetPermissionVPToValidated(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	otherAddr := sdk.AccAddress([]byte("other_user")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()

	futureTime := now.Add(365 * 24 * time.Hour)
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create validator perm
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}

	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// 1. Test with new perm (not renewal case)
	t.Run("Valid new perm validation", func(t *testing.T) {
		// Create a new perm in PENDING state
		newPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		// Set perm to validated
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      newPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     0, // Default no discount
			VerificationFeeDiscount: 0, // Default no discount
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify perm was updated correctly
		updatedPerm, err := k.GetParticipantByID(sdkCtx, newPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_VALIDATED, updatedPerm.OpState)
		require.Equal(t, msg.ValidationFees, updatedPerm.ValidationFees)
		require.Equal(t, msg.IssuanceFees, updatedPerm.IssuanceFees)
		require.Equal(t, msg.VerificationFees, updatedPerm.VerificationFees)
		require.Equal(t, msg.IssuanceFeeDiscount, updatedPerm.IssuanceFeeDiscount)
		require.Equal(t, msg.VerificationFeeDiscount, updatedPerm.VerificationFeeDiscount)
		require.NotNil(t, updatedPerm.EffectiveFrom)
		require.Equal(t, now.Unix(), updatedPerm.EffectiveFrom.Unix()) // First time: set to now
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, futureTime.Unix(), updatedPerm.EffectiveUntil.Unix())
		require.Equal(t, msg.OpSummaryDigest, updatedPerm.OpSummaryDigest)
		// Execution assertions
		require.NotNil(t, updatedPerm.Modified)
		require.Equal(t, now.Unix(), updatedPerm.Modified.Unix())
		require.NotNil(t, updatedPerm.OpLastStateChange)
		require.Equal(t, now.Unix(), updatedPerm.OpLastStateChange.Unix())
		require.Equal(t, uint64(0), updatedPerm.OpCurrentFees)    // Reset to 0
		require.Equal(t, uint64(0), updatedPerm.OpCurrentDeposit) // Reset to 0
	})

	// 2. Test renewal case - perm already has EffectiveFrom
	t.Run("Renewal perm validation", func(t *testing.T) {
		renewalAddr := sdk.AccAddress([]byte("renewal_creator")).String()
		// Create a perm that already has EffectiveFrom set (renewal)
		effectiveFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
		currentEffectiveUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(renewalAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffectiveUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		renewalPermID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		// Set perm to validated with same fees
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      renewalPermID,
			ValidationFees:          10, // Same as existing
			IssuanceFees:            5,  // Same as existing
			VerificationFees:        3,  // Same as existing
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-renewalDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify perm was updated correctly
		updatedPerm, err := k.GetParticipantByID(sdkCtx, renewalPermID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_VALIDATED, updatedPerm.OpState)
		// Fees should remain unchanged (renewal doesn't overwrite)
		require.Equal(t, renewalPerm.ValidationFees, updatedPerm.ValidationFees)
		require.Equal(t, renewalPerm.IssuanceFees, updatedPerm.IssuanceFees)
		require.Equal(t, renewalPerm.VerificationFees, updatedPerm.VerificationFees)
		// EffectiveFrom should NOT change on renewal
		require.Equal(t, effectiveFrom.Unix(), updatedPerm.EffectiveFrom.Unix())
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, futureTime.Unix(), updatedPerm.EffectiveUntil.Unix())
	})

	// 3. Test validation error - Invalid Participant ID
	t.Run("Invalid Participant ID", func(t *testing.T) {
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation: validatorAddr,
			Operator:    validatorAddr,
			Id:          9999, // Non-existent ID
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm not found")
		require.Nil(t, resp)
	})

	// 4. Test validation error - Not in PENDING state
	t.Run("Not in PENDING state", func(t *testing.T) {
		// Create a perm that's not in PENDING state
		notPendingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED, // Not PENDING
		}
		notPendingPermID, err := k.CreatePermission(sdkCtx, notPendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation: validatorAddr,
			Operator:    validatorAddr,
			Id:          notPendingPermID,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm must be in PENDING state")
		require.Nil(t, resp)
	})

	// 5. Test validation error - Wrong validator
	t.Run("Wrong validator", func(t *testing.T) {
		// Create a new perm in PENDING state
		pendingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		pendingPermID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation: otherAddr, // Not the validator
			Operator:    otherAddr,
			Id:          pendingPermID,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority must be validator participant authority")
		require.Nil(t, resp)
	})

	// 6. Test validation error - HOLDER with digest SRI
	t.Run("HOLDER type with digest SRI", func(t *testing.T) {
		// Create a HOLDER perm in PENDING state
		holderPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_HOLDER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		holderPermID, err := k.CreatePermission(sdkCtx, holderPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      holderPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			OpSummaryDigest:         "sha384-someDigest", // Should be empty for HOLDER
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "op_summary_digest must be null for HOLDER type")
		require.Nil(t, resp)
	})

	// 7. Test discount validation - ISSUER_GRANTOR with valid discount
	t.Run("ISSUER_GRANTOR with valid discount", func(t *testing.T) {
		// Create ISSUER_GRANTOR perm in PENDING state
		grantorPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      grantorPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     5000, // 50% discount
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, grantorPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), updatedPerm.IssuanceFeeDiscount)
	})

	// 8. Test discount validation - ISSUER in GRANTOR mode with discount within validator's limit
	t.Run("ISSUER in GRANTOR mode with valid discount", func(t *testing.T) {
		// First create a validator with a discount
		validatorWithDiscount := types.Participant{
			SchemaId:            1,
			Role:                types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:       trkKeeper.RegisterCorp(validatorAddr),
			Created:             &now,
			Adjusted:            &now,
			Modified:            &now,
			OpState:             types.OnboardingState_VALIDATED,
			IssuanceFeeDiscount: 7000,      // 70% discount
			EffectiveFrom:       &pastTime, // Required for ACTIVE state
		}
		validatorWithDiscountID, err := k.CreatePermission(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		// Create ISSUER perm with this validator
		issuerPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorWithDiscountID,
			OpState:                types.OnboardingState_PENDING,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		// Can set discount up to validator's discount (7000)
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     5000, // 50% discount (within validator's 70%)
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, issuerPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), updatedPerm.IssuanceFeeDiscount)
	})

	// 9. Test discount validation - ISSUER in GRANTOR mode exceeding validator's discount
	t.Run("ISSUER in GRANTOR mode exceeding validator discount", func(t *testing.T) {
		// Create ISSUER perm with validator that has 50% discount
		validatorWithDiscount := types.Participant{
			SchemaId:            1,
			Role:                types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:       trkKeeper.RegisterCorp(validatorAddr),
			Created:             &now,
			Adjusted:            &now,
			Modified:            &now,
			OpState:             types.OnboardingState_VALIDATED,
			IssuanceFeeDiscount: 5000,      // 50% discount
			EffectiveFrom:       &pastTime, // Required for ACTIVE state
		}
		validatorWithDiscountID, err := k.CreatePermission(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		issuerPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorWithDiscountID,
			OpState:                types.OnboardingState_PENDING,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		// Try to set discount exceeding validator's discount
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     6000, // 60% discount (exceeds validator's 50%)
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed validator's discount")
		require.Nil(t, resp)
	})

	// 10. Test discount validation - discount exceeds maximum
	t.Run("Discount exceeds maximum", func(t *testing.T) {
		grantorPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      grantorPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     10001, // Exceeds maximum of 10000
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed")
		require.Nil(t, resp)
	})

	// 11. Test renewal with discount - must match existing discount
	t.Run("Renewal with discount must match existing", func(t *testing.T) {
		effectiveFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
		renewalPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(otherAddr), // Use different authority to avoid overlap with test 7
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
			IssuanceFeeDiscount:    3000, // 30% discount set initially
		}
		renewalPermID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		// Try to change discount during renewal
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      renewalPermID,
			ValidationFees:          10, // Must match
			IssuanceFees:            5,  // Must match
			VerificationFees:        3,  // Must match
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     4000, // Different from existing 3000
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be changed during renewal")
		require.Nil(t, resp)

		// Try with matching discount
		msg.IssuanceFeeDiscount = 3000 // Match existing
		resp, err = ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, renewalPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(3000), updatedPerm.IssuanceFeeDiscount)
	})

	// 12. Test ECOSYSTEM mode - ISSUER with discount
	t.Run("ISSUER in ECOSYSTEM mode with discount", func(t *testing.T) {
		// Create schema with ECOSYSTEM mode
		csKeeper.CreateMockCredentialSchema(2,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS)

		// Create ECOSYSTEM validator
		ecosystemValidator := types.Participant{
			SchemaId:      2,
			Role:          types.ParticipantRole_ECOSYSTEM,
			CorporationId: trkKeeper.RegisterCorp(validatorAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime, // Required for ACTIVE state
		}
		ecosystemValidatorID, err := k.CreatePermission(sdkCtx, ecosystemValidator)
		require.NoError(t, err)

		// Create ISSUER perm with ECOSYSTEM validator
		issuerPerm := types.Participant{
			SchemaId:               2,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: ecosystemValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     8000, // 80% discount (allowed in ECOSYSTEM mode)
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, issuerPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(8000), updatedPerm.IssuanceFeeDiscount)
	})

	// 13. Test effective_until <= now (first time) should fail
	t.Run("effective_until must be greater than now for first time", func(t *testing.T) {
		euAddr := sdk.AccAddress([]byte("eu_now_creator")).String()
		pendingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(euAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		pastEffUntil := now.Add(-1 * time.Hour) // in the past
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &pastEffUntil,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be greater than current timestamp")
		require.Nil(t, resp)
	})

	// 14. Test effective_until > op_exp should fail
	t.Run("effective_until must be lower or equal to op_exp", func(t *testing.T) {
		// Create schema with validity period so vpExp is calculated
		csKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                             3,
			IssuerOnboardingMode:           cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			VerifierOnboardingMode:         cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			IssuerValidationValidityPeriod: 30, // 30 days
		})

		// Create validator for schema 3
		vpAddr := sdk.AccAddress([]byte("op_exp_validator")).String()
		vpValidator := types.Participant{
			SchemaId:      3,
			Role:          types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(vpAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vpValidatorID, err := k.CreatePermission(sdkCtx, vpValidator)
		require.NoError(t, err)

		vpTestAddr := sdk.AccAddress([]byte("op_exp_creator")).String()
		pendingPerm := types.Participant{
			SchemaId:               3,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(vpTestAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vpValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		// vpExp will be now + 30 days. Set effective_until to now + 60 days (beyond vpExp)
		farFuture := now.Add(60 * 24 * time.Hour)
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      vpAddr,
			Operator:         vpAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &farFuture,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be lower or equal to op_exp")
		require.Nil(t, resp)
	})

	// 15. Test effective_until nil resolves to vpExp
	t.Run("effective_until nil resolves to op_exp", func(t *testing.T) {
		// Schema 3 already has 30-day validity period from test 14
		vpAddr := sdk.AccAddress([]byte("op_exp_validator")).String()
		// Find the validator perm ID for schema 3
		vpNilAddr := sdk.AccAddress([]byte("vp_nil_creator")).String()

		// Create validator for schema 3 (separate to avoid overlap)
		vpValidator2 := types.Participant{
			SchemaId:      3,
			Role:          types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(vpAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vpValidator2ID, err := k.CreatePermission(sdkCtx, vpValidator2)
		require.NoError(t, err)

		pendingPerm := types.Participant{
			SchemaId:               3,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(vpNilAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vpValidator2ID,
			OpState:                types.OnboardingState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      vpAddr,
			Operator:         vpAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   nil, // nil should resolve to vpExp
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, permID)
		require.NoError(t, err)
		// effective_until should equal vpExp (now + 30 days)
		expectedVpExp := now.AddDate(0, 0, 30)
		require.NotNil(t, updatedPerm.OpExp)
		require.Equal(t, expectedVpExp.Unix(), updatedPerm.OpExp.Unix())
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, expectedVpExp.Unix(), updatedPerm.EffectiveUntil.Unix())
	})

	// 16. Test renewal effective_until must be greater than current effective_until
	t.Run("Renewal effective_until must be greater than current", func(t *testing.T) {
		renewAddr := sdk.AccAddress([]byte("renew_eu_creator")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(30 * 24 * time.Hour) // 30 days in future
		renewalPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(renewAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		// Try with effective_until <= current effective_until
		smallerEffUntil := now.Add(10 * 24 * time.Hour)
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &smallerEffUntil,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be greater than current effective_until")
		require.Nil(t, resp)
	})

	// 17. Test renewal validation_fees mismatch
	t.Run("Renewal validation_fees must match", func(t *testing.T) {
		rvfAddr := sdk.AccAddress([]byte("ren_valfees_addr")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rvfAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   20, // Different from existing 10
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation_fees cannot be changed during renewal")
		require.Nil(t, resp)
	})

	// 18. Test renewal issuance_fees mismatch
	t.Run("Renewal issuance_fees must match", func(t *testing.T) {
		rifAddr := sdk.AccAddress([]byte("ren_issfees_addr")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rifAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     99, // Different from existing 5
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "issuance_fees cannot be changed during renewal")
		require.Nil(t, resp)
	})

	// 19. Test renewal verification_fees mismatch
	t.Run("Renewal verification_fees must match", func(t *testing.T) {
		rvAddr := sdk.AccAddress([]byte("ren_verfees_addr")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rvAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 99, // Different from existing 3
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "verification_fees cannot be changed during renewal")
		require.Nil(t, resp)
	})

	// 20. Test overlap - existing perm never expires (nil effective_until)
	t.Run("Overlap with never-expiring permission", func(t *testing.T) {
		overlapAddr := sdk.AccAddress([]byte("overlap_never_addr")).String()
		// Create an existing validated perm with nil effective_until (never expires)
		existingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         nil, // Never expires
		}
		_, err := k.CreatePermission(sdkCtx, existingPerm)
		require.NoError(t, err)

		// Try to create a new validated perm with same (schema, type, validator, authority)
		newPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               newPermID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap check failed")
		require.Contains(t, err.Error(), "never expires")
		require.Nil(t, resp)
	})

	// 21. Test overlap - existing perm's effective_until after new perm's effective_from
	t.Run("Overlap with active permission time range", func(t *testing.T) {
		overlapAddr2 := sdk.AccAddress([]byte("overlap_range_addr")).String()
		// Create an existing validated perm that's still active (effective_until in the future)
		existingEffUntil := now.Add(30 * 24 * time.Hour) // 30 days from now
		existingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr2),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         &existingEffUntil,
		}
		_, err := k.CreatePermission(sdkCtx, existingPerm)
		require.NoError(t, err)

		// Try to validate a new perm — effective_from will be set to now, which is before existing's effective_until
		newPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr2),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               newPermID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap check failed")
		require.Nil(t, resp)
	})

	// 22. Test validator perm not active (revoked)
	t.Run("Validator perm is revoked", func(t *testing.T) {
		revokedTime := now.Add(-1 * time.Hour)
		revokedValidator := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(validatorAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
			Revoked:       &revokedTime,
		}
		revokedValidatorID, err := k.CreatePermission(sdkCtx, revokedValidator)
		require.NoError(t, err)

		rvAddr := sdk.AccAddress([]byte("revoked_val_addr")).String()
		pendingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rvAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: revokedValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	// 23. Test with OpCurrentFees and OpCurrentDeposit > 0 (execution: fee transfer + trust deposit)
	t.Run("Execution with fees and trust deposit", func(t *testing.T) {
		feeAddr := sdk.AccAddress([]byte("fee_exec_creator")).String()
		newPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(feeAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          100, // Has fees to transfer
			OpCurrentDeposit:       50,  // Has deposit to transfer
		}
		permID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               permID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, permID)
		require.NoError(t, err)
		require.Equal(t, uint64(0), updatedPerm.OpCurrentFees)       // Reset to 0
		require.Equal(t, uint64(0), updatedPerm.OpCurrentDeposit)    // Reset to 0
		require.Equal(t, uint64(50), updatedPerm.OpValidatorDeposit) // Accumulated
	})

	// 24. Test VERIFIER_GRANTOR with verification_fee_discount
	t.Run("VERIFIER_GRANTOR with verification_fee_discount", func(t *testing.T) {
		// Create schema with GRANTOR_VALIDATION for verifier mode
		csKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                     4,
			IssuerOnboardingMode:   cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			VerifierOnboardingMode: cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		})

		vgAddr := sdk.AccAddress([]byte("ver_grantor_vali")).String()
		vgValidator := types.Participant{
			SchemaId:      4,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(vgAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgValidatorID, err := k.CreatePermission(sdkCtx, vgValidator)
		require.NoError(t, err)

		// Create VERIFIER_GRANTOR perm (can set its own verification_fee_discount)
		vgPerm := types.Participant{
			SchemaId:               4,
			Role:                   types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(sdk.AccAddress([]byte("vg_perm_creator")).String()),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vgValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		vgPermID, err := k.CreatePermission(sdkCtx, vgPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             vgAddr,
			Operator:                vgAddr,
			Id:                      vgPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 6000, // 60% discount
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetParticipantByID(sdkCtx, vgPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(6000), updatedPerm.VerificationFeeDiscount)
	})

	// 25. Test VERIFIER in GRANTOR mode exceeding validator's verification_fee_discount
	t.Run("VERIFIER exceeding validator verification_fee_discount", func(t *testing.T) {
		// Create validator with verification_fee_discount
		vgAddr2 := sdk.AccAddress([]byte("vg_disc_validato")).String()
		vgValidator2 := types.Participant{
			SchemaId:                4,
			Role:                    types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId:           trkKeeper.RegisterCorp(vgAddr2),
			Created:                 &now,
			Adjusted:                &now,
			Modified:                &now,
			OpState:                 types.OnboardingState_VALIDATED,
			VerificationFeeDiscount: 5000, // 50% discount
			EffectiveFrom:           &pastTime,
		}
		vgValidator2ID, err := k.CreatePermission(sdkCtx, vgValidator2)
		require.NoError(t, err)

		verPerm := types.Participant{
			SchemaId:               4,
			Role:                   types.ParticipantRole_VERIFIER,
			CorporationId:          trkKeeper.RegisterCorp(sdk.AccAddress([]byte("ver_exceed_addr")).String()),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vgValidator2ID,
			OpState:                types.OnboardingState_PENDING,
		}
		verPermID, err := k.CreatePermission(sdkCtx, verPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             vgAddr2,
			Operator:                vgAddr2,
			Id:                      verPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 7000, // 70% exceeds validator's 50%
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed validator's discount")
		require.Nil(t, resp)
	})

	// 26. Test overlap check skips revoked permissions
	t.Run("Overlap check skips revoked permissions", func(t *testing.T) {
		skipAddr := sdk.AccAddress([]byte("overlap_skip_addr")).String()
		revokedTime := now.Add(-1 * time.Hour)
		// Create a revoked perm (should be skipped in overlap check)
		revokedPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(skipAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         nil, // Never expires, but it's revoked
			Revoked:                &revokedTime,
		}
		_, err := k.CreatePermission(sdkCtx, revokedPerm)
		require.NoError(t, err)

		// This new perm should NOT conflict with the revoked one
		newPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(skipAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               newPermID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 27. Test verification_fee_discount exceeds maximum
	t.Run("Verification_fee_discount exceeds maximum", func(t *testing.T) {
		vfdAddr := sdk.AccAddress([]byte("vfd_max_creator")).String()
		grantorPerm := types.Participant{
			SchemaId:      4,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(vfdAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			ValidatorParticipantId: func() uint64 {
				// Reuse a schema 4 validator
				vAddr := sdk.AccAddress([]byte("vfd_max_validato")).String()
				v := types.Participant{
					SchemaId:      4,
					Role:          types.ParticipantRole_VERIFIER_GRANTOR,
					CorporationId: trkKeeper.RegisterCorp(vAddr),
					Created:       &now,
					Adjusted:      &now,
					Modified:      &now,
					OpState:       types.OnboardingState_VALIDATED,
					EffectiveFrom: &pastTime,
				}
				id, _ := k.CreatePermission(sdkCtx, v)
				return id
			}(),
			OpState: types.OnboardingState_PENDING,
		}
		grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             sdk.AccAddress([]byte("vfd_max_validato")).String(),
			Operator:                sdk.AccAddress([]byte("vfd_max_validato")).String(),
			Id:                      grantorPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 10001, // Exceeds maximum
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed")
		require.Nil(t, resp)
	})

	// 28. Test renewal discount for verification_fee_discount must match
	t.Run("Renewal verification_fee_discount must match existing", func(t *testing.T) {
		rvdAddr := sdk.AccAddress([]byte("ren_vfd_creator")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Participant{
			SchemaId:      4,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(rvdAddr),
			Created:       &now,
			Adjusted:      &now,
			Modified:      &now,
			ValidatorParticipantId: func() uint64 {
				vAddr := sdk.AccAddress([]byte("rvd_validator_ad")).String()
				v := types.Participant{
					SchemaId:      4,
					Role:          types.ParticipantRole_VERIFIER_GRANTOR,
					CorporationId: trkKeeper.RegisterCorp(vAddr),
					Created:       &now,
					Adjusted:      &now,
					Modified:      &now,
					OpState:       types.OnboardingState_VALIDATED,
					EffectiveFrom: &pastTime,
				}
				id, _ := k.CreatePermission(sdkCtx, v)
				return id
			}(),
			OpState:                 types.OnboardingState_PENDING,
			EffectiveFrom:           &effectiveFrom,
			EffectiveUntil:          &currentEffUntil,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			VerificationFeeDiscount: 4000, // Existing 40%
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             sdk.AccAddress([]byte("rvd_validator_ad")).String(),
			Operator:                sdk.AccAddress([]byte("rvd_validator_ad")).String(),
			Id:                      permID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			OpSummaryDigest:         "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 6000, // Different from existing 4000
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "verification_fee_discount cannot be changed during renewal")
		require.Nil(t, resp)
	})
}

// Test AUTHZ-CHECK failure for SetParticipantOPToValidated
func TestSetPermissionVPToValidated_AuthzCheckFailure(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	operatorAddr := sdk.AccAddress([]byte("test_operator")).String()
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	// Create validator perm
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create perm to validate
	pendingPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creatorAddr),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_PENDING,
	}
	permID, err := k.CreatePermission(sdkCtx, pendingPerm)
	require.NoError(t, err)

	// Set delegation keeper to return error
	delKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

	msg := &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         operatorAddr,
		Id:               permID,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveUntil:   &futureTime,
		OpSummaryDigest:  "sha384-validDigest",
	}

	resp, err := ms.SetParticipantOPToValidated(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
	require.Contains(t, err.Error(), "operator not authorized")
	require.Nil(t, resp)

	// Reset to allow and verify it works
	delKeeper.ErrToReturn = nil

	// Need to use validatorAddr as authority (not creatorAddr) since validator perm authority check
	resp, err = ms.SetParticipantOPToValidated(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// TestMsgServerCreateRootPermission is superseded by TestCreateRootPermission which has
// comprehensive coverage of all spec v4 checks including overlap and AUTHZ.
// Keeping as a simple smoke test with updated field names.
func TestMsgServerCreateRootPermission(t *testing.T) {
	k, ms, mockCsKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	authority := sdk.AccAddress([]byte("test_creator________")).String()
	operator := authority
	validDid := "did:example:123456789abcdefghi"

	trID := trkKeeper.CreateMockEcosystem(authority, validDid)
	mockCsKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	blockTime := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	now := sdkCtx.BlockTime()
	futureTime := now.Add(1 * time.Hour)
	farFuture := now.Add(24 * time.Hour)

	// Valid creation
	resp, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: authority, Operator: operator,
		SchemaId: 1, Did: validDid,
		ValidationFees: 100, IssuanceFees: 50, VerificationFees: 25,
		EffectiveFrom: &futureTime, EffectiveUntil: &farFuture,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, uint64(1), resp.Id)

	perm, err := k.GetParticipantByID(sdkCtx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, uint64(1), perm.SchemaId)
	require.Equal(t, validDid, perm.Did)
	require.NotZero(t, perm.CorporationId)
	// [MOD-PERM-MSG-7-3] spec v4 draft 13: perm.type is hardcoded to ECOSYSTEM.
	require.Equal(t, types.ParticipantRole_ECOSYSTEM, perm.Role)
	require.Equal(t, uint64(100), perm.ValidationFees)
	require.Equal(t, uint64(50), perm.IssuanceFees)
	require.Equal(t, uint64(25), perm.VerificationFees)
	require.Equal(t, uint64(0), perm.Deposit)
	require.NotNil(t, perm.Created)
	require.NotNil(t, perm.Modified)
}

func TestCancelPermissionVPLastRequest(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	otherAddr := sdk.AccAddress([]byte("other_user")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()

	// Create validator perm
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// [MOD-PERM-MSG-6-3] Spec v4 draft 13: when op_exp is null (never validated),
	// set op_state to TERMINATED. The permission row is retained.
	t.Run("Valid cancellation - never validated before", func(t *testing.T) {
		neverAddr := sdk.AccAddress([]byte("never_val_cancel")).String()
		neverValidatedPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(neverAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		permID, err := k.CreatePermission(sdkCtx, neverValidatedPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: neverAddr,
			Operator:    neverAddr,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Participant is retained and transitioned to TERMINATED.
		got, err := k.GetParticipantByID(sdkCtx, permID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_TERMINATED, got.OpState)
	})

	// 2. Valid cancellation - previously validated (renewal: EffectiveFrom set → VALIDATED)
	t.Run("Valid cancellation - previously validated", func(t *testing.T) {
		prevAddr := sdk.AccAddress([]byte("prev_val_cancel")).String()
		pastTime := now.Add(-1 * time.Hour)
		futureTime := now.Add(24 * time.Hour)
		previouslyValidatedPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(prevAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			OpExp:                  &futureTime, // Has a previous validation
			EffectiveFrom:          &pastTime,   // Renewal: was previously activated
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		permID, err := k.CreatePermission(sdkCtx, previouslyValidatedPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: prevAddr,
			Operator:    prevAddr,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, permID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_VALIDATED, perm.OpState)
		require.Equal(t, uint64(0), perm.OpCurrentFees)
		require.Equal(t, uint64(0), perm.OpCurrentDeposit)
	})

	// 3. Invalid - perm not found
	t.Run("Invalid - perm not found", func(t *testing.T) {
		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: creator,
			Operator:    creator,
			Id:          9999,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm not found")
		require.Nil(t, resp)
	})

	// 4. Invalid - wrong authority
	t.Run("Invalid - wrong authority", func(t *testing.T) {
		wrongAuthPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, wrongAuthPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: otherAddr, // Not the perm authority
			Operator:    otherAddr,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not the participant authority")
		require.Nil(t, resp)
	})

	// 5. Invalid - not in PENDING state
	t.Run("Invalid - not in PENDING state", func(t *testing.T) {
		notPendingPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
		}
		permID, err := k.CreatePermission(sdkCtx, notPendingPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: creator,
			Operator:    creator,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm must be in PENDING state")
		require.Nil(t, resp)
	})

	// 6. Invalid - slashed and not repaid
	t.Run("Invalid - slashed and not repaid", func(t *testing.T) {
		slashedTime := now.Add(-1 * time.Hour)
		slashedPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			Slashed:                &slashedTime, // Slashed
			// Repaid is nil (not repaid)
		}
		permID, err := k.CreatePermission(sdkCtx, slashedPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: creator,
			Operator:    creator,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "slashed and not repaid")
		require.Nil(t, resp)
	})

	// 7. Valid - slashed but repaid (allowed), first-time VP → perm deleted
	t.Run("Valid - slashed and repaid is allowed", func(t *testing.T) {
		repaidAddr := sdk.AccAddress([]byte("repaid_cancel_ad")).String()
		slashedTime := now.Add(-2 * time.Hour)
		repaidTime := now.Add(-1 * time.Hour)
		repaidPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(repaidAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			Slashed:                &slashedTime,
			Repaid:                 &repaidTime, // Repaid
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		permID, err := k.CreatePermission(sdkCtx, repaidPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: repaidAddr,
			Operator:    repaidAddr,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// [MOD-PERM-MSG-6-3] Never-validated permission transitions to TERMINATED; row retained.
		got, err := k.GetParticipantByID(sdkCtx, permID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_TERMINATED, got.OpState)
	})

	// 8. Valid cancellation with zero fees (no transfer needed)
	t.Run("Valid cancellation with zero fees", func(t *testing.T) {
		zeroFeesAddr := sdk.AccAddress([]byte("zero_fees_cancel")).String()
		zeroFeesPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(zeroFeesAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		permID, err := k.CreatePermission(sdkCtx, zeroFeesPerm)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: zeroFeesAddr,
			Operator:    zeroFeesAddr,
			Id:          permID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestCancelPermissionVPLastRequest_AuthzCheckFailure tests AUTHZ-CHECK for CancelParticipantOPLastRequest
func TestCancelPermissionVPLastRequest_AuthzCheckFailure(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()
	operatorAddr := sdk.AccAddress([]byte("test_operator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()

	now := sdkCtx.BlockTime()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	pendingPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creatorAddr),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_PENDING,
	}
	permID, err := k.CreatePermission(sdkCtx, pendingPerm)
	require.NoError(t, err)

	// Set delegation keeper to return error
	delKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

	msg := &types.MsgCancelParticipantOPLastRequest{
		Corporation: creatorAddr,
		Operator:    operatorAddr,
		Id:          permID,
	}

	resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
	require.Contains(t, err.Error(), "operator not authorized")
	require.Nil(t, resp)

	// Reset and verify it works
	delKeeper.ErrToReturn = nil
	resp, err = ms.CancelParticipantOPLastRequest(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// TestAdjustPermission tests the SetParticipantEffectiveUntil message server function
func TestAdjustPermission(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority__")).String()
	operatorAddr := sdk.AccAddress([]byte("test_operator___")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator__")).String()
	ecosystemAddr := sdk.AccAddress([]byte("trust_registry__")).String()
	wrongAddr := sdk.AccAddress([]byte("wrong_authority_")).String()

	// Create distinct mock credential schemas to avoid overlap between test permissions.
	// Each permission uses a unique schema_id so the overlap check doesn't fire across test cases.
	for i := uint64(1); i <= 10; i++ {
		csKeeper.CreateMockCredentialSchema(i,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	}

	now := sdkCtx.BlockTime()
	currentEffectiveUntil := now.Add(30 * 24 * time.Hour) // 30 days in the future
	futureVpExp := now.Add(365 * 24 * time.Hour)          // 1 year in the future
	pastTime := now.Add(-1 * time.Hour)                   // Set effective_from to past to make it ACTIVE

	// Create validator perm (ISSUER_GRANTOR) — schema 1
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a VP managed perm to adjust — schema 2
	applicantPerm := types.Participant{
		SchemaId:               2,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         &currentEffectiveUntil,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create an ECOSYSTEM perm — schema 3
	ecosystemPerm := types.Participant{
		SchemaId:       3,
		Role:           types.ParticipantRole_ECOSYSTEM,
		CorporationId:  trkKeeper.RegisterCorp(ecosystemAddr),
		Created:        &now,
		Adjusted:       &now,
		Modified:       &now,
		EffectiveUntil: &currentEffectiveUntil,
		OpState:        types.OnboardingState_VALIDATED,
		EffectiveFrom:  &pastTime,
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create a perm for the "wrong authority" test — schema 4
	wrongAuthTestPerm := types.Participant{
		SchemaId:               4,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         &currentEffectiveUntil,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	wrongAuthTestPermID, err := k.CreatePermission(sdkCtx, wrongAuthTestPerm)
	require.NoError(t, err)

	// Create a perm with NULL effective_until — schema 5
	nullEffectiveUntilPerm := types.Participant{
		SchemaId:               5,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         nil,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	nullEffectiveUntilPermID, err := k.CreatePermission(sdkCtx, nullEffectiveUntilPerm)
	require.NoError(t, err)

	// Create an ecosystem perm with NULL effective_until — schema 6
	nullEffectiveUntilEcosystemPerm := types.Participant{
		SchemaId:       6,
		Role:           types.ParticipantRole_ECOSYSTEM,
		CorporationId:  trkKeeper.RegisterCorp(ecosystemAddr),
		Created:        &now,
		Adjusted:       &now,
		Modified:       &now,
		EffectiveUntil: nil,
		OpState:        types.OnboardingState_VALIDATED,
		EffectiveFrom:  &pastTime,
	}
	nullEffectiveUntilEcosystemPermID, err := k.CreatePermission(sdkCtx, nullEffectiveUntilEcosystemPerm)
	require.NoError(t, err)

	// Create perm for past effective_until test — schema 7
	nullEffUntilPastTestPerm := types.Participant{
		SchemaId:               7,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         nil,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	nullEffUntilPastTestPermID, err := k.CreatePermission(sdkCtx, nullEffUntilPastTestPerm)
	require.NoError(t, err)

	// Create perm for reduce effective_until test — schema 8
	reducePerm := types.Participant{
		SchemaId:               8,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Modified:               &now,
		EffectiveUntil:         &currentEffectiveUntil, // 30 days
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	reducePermID, err := k.CreatePermission(sdkCtx, reducePerm)
	require.NoError(t, err)

	newEffectiveUntil := now.Add(60 * 24 * time.Hour)     // 60 days in the future
	pastEffectiveUntil := now.Add(-1 * 24 * time.Hour)    // 1 day in the past
	tooFarEffectiveUntil := now.Add(500 * 24 * time.Hour) // Past VP expiration
	equalToNowEffectiveUntil := now                       // Equal to now (should fail)
	reducedEffectiveUntil := now.Add(15 * 24 * time.Hour) // 15 days — less than current 30 days

	testCases := []struct {
		name       string
		msg        *types.MsgSetParticipantEffectiveUntil
		expectErr  bool
		errMessage string
	}{
		{
			name: "Valid adjustment by validator authority (VP managed)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Valid adjustment by ecosystem authority",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    ecosystemAddr,
				Operator:       operatorAddr,
				Id:             ecosystemPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             9999,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "permission not found",
		},
		{
			name: "Invalid - effective_until in the past",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &pastEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current timestamp",
		},
		{
			name: "Invalid - effective_until equal to now",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &equalToNowEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current timestamp",
		},
		{
			name: "Invalid - effective_until beyond validation expiration (VP managed)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &tooFarEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until cannot be after validation expiration",
		},
		{
			name: "Invalid - wrong authority (VP managed)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    wrongAddr,
				Operator:       operatorAddr,
				Id:             wrongAuthTestPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "authority is not the validator participant authority",
		},
		{
			name: "Valid - adjust permission with NULL effective_until (VP managed)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             nullEffectiveUntilPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Valid - adjust permission with NULL effective_until (ecosystem)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    ecosystemAddr,
				Operator:       operatorAddr,
				Id:             nullEffectiveUntilEcosystemPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Invalid - effective_until in the past (NULL current effective_until)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             nullEffUntilPastTestPermID,
				EffectiveUntil: &pastEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current timestamp",
		},
		{
			name: "Valid - reduce effective_until (v4 allows reduction)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             reducePermID,
				EffectiveUntil: &reducedEffectiveUntil,
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SetParticipantEffectiveUntil(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm was adjusted
				perm, err := k.GetParticipantByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.EffectiveUntil.Unix(), perm.EffectiveUntil.Unix())
				require.NotNil(t, perm.Adjusted)
				require.NotNil(t, perm.Modified)
			}
		})
	}
}

// TestRevokePermission tests the RevokeParticipant message server function
func TestRevokePermission(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority__")).String()
	operatorAddr := sdk.AccAddress([]byte("test_operator___")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator__")).String()
	wrongAddr := sdk.AccAddress([]byte("wrong_authority_")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	csKeeper.CreateMockCredentialSchema(2,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create validator perm — schema 1
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm to revoke — schema 2
	applicantPerm := types.Participant{
		SchemaId:               2,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create another perm for the wrong-authority test — schema 2
	wrongAuthPerm := types.Participant{
		SchemaId:               2,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	wrongAuthPermID, err := k.CreatePermission(sdkCtx, wrongAuthPerm)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgRevokeParticipant
		expectErr  bool
		errMessage string
	}{
		{
			name: "Valid revocation by validator ancestor",
			msg: &types.MsgRevokeParticipant{
				Corporation: validatorAddr,
				Operator:    operatorAddr,
				Id:          applicantPermID,
			},
			expectErr: false,
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgRevokeParticipant{
				Corporation: validatorAddr,
				Operator:    operatorAddr,
				Id:          9999,
			},
			expectErr:  true,
			errMessage: "permission not found",
		},
		{
			name: "Invalid - wrong authority (not validator, not self, not TR controller)",
			msg: &types.MsgRevokeParticipant{
				Corporation: wrongAddr,
				Operator:    operatorAddr,
				Id:          wrongAuthPermID,
			},
			expectErr:  true,
			errMessage: "authority is not authorized to revoke this participant",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RevokeParticipant(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm was revoked
				perm, err := k.GetParticipantByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, perm.Revoked)
			}
		})
	}
}

func TestCreateOrUpdateParticipantSession(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	otherAuthority := sdk.AccAddress([]byte("other_authority")).String()
	otherOperator := sdk.AccAddress([]byte("other_operator")).String()
	sessionUUID := uuid.New().String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past to make it ACTIVE

	// Create trust registry / ecosystem perm
	trustPerm := types.Participant{
		SchemaId:         1,
		Role:             types.ParticipantRole_ECOSYSTEM,
		CorporationId:    trkKeeper.RegisterCorp(authority),
		Created:          &now,
		Adjusted:         &now,
		Modified:         &now,
		OpState:          types.OnboardingState_VALIDATED,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveFrom:    &pastTime,
	}
	trustPermID, err := k.CreatePermission(sdkCtx, trustPerm)
	require.NoError(t, err)

	// Create issuer perm with VsOperator and VsOperatorAuthzEnabled
	issuerPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		VsOperatorAuthzEnabled: true,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Create issuer perm with VsOperatorAuthzEnabled = false
	issuerPermNoAuthz := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		VsOperatorAuthzEnabled: false,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerPermNoAuthzID, err := k.CreatePermission(sdkCtx, issuerPermNoAuthz)
	require.NoError(t, err)

	// Create issuer perm with different vs_operator
	issuerPermDiffOp := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             otherOperator,
		VsOperatorAuthzEnabled: true,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerPermDiffOpID, err := k.CreatePermission(sdkCtx, issuerPermDiffOp)
	require.NoError(t, err)

	// Create issuer perm with different authority
	issuerPermDiffAuth := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(otherAuthority),
		VsOperator:             operator,
		VsOperatorAuthzEnabled: true,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerPermDiffAuthID, err := k.CreatePermission(sdkCtx, issuerPermDiffAuth)
	require.NoError(t, err)

	// Create verifier perm
	verifierPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_VERIFIER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		VsOperatorAuthzEnabled: true,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	verifierPermID, err := k.CreatePermission(sdkCtx, verifierPerm)
	require.NoError(t, err)

	// Create agent perm
	agentPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	// Create wallet agent perm
	walletAgentPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	walletAgentPermID, err := k.CreatePermission(sdkCtx, walletAgentPerm)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgCreateOrUpdateParticipantSession
		setupErr   error // set on delKeeper.ErrToReturn before test
		expectErr  bool
		errMessage string
	}{
		{
			name: "Happy path with issuer",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       sessionUUID,
				IssuerParticipantId:      issuerPermID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr: false,
		},
		{
			name: "Happy path with verifier",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      0,
				VerifierParticipantId:    verifierPermID,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr: false,
		},
		{
			name: "AUTHZ check failure",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerPermID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			setupErr:   fmt.Errorf("operator not authorized"),
			expectErr:  true,
			errMessage: "VS operator authorization check failed",
		},
		// Note: Invalid UUID is caught by ValidateBasic at SDK level, not in the handler
		{
			name: "Both perms missing",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      0,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "at least one of issuer_perm_id or verifier_perm_id must be provided",
		},
		{
			name: "Issuer perm not found",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      9999,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "issuer permission not found",
		},
		{
			name: "Issuer wrong type",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      trustPermID, // ECOSYSTEM type, not ISSUER
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "issuer permission must be ISSUER type",
		},
		{
			name: "Issuer vs_operator mismatch",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator, // does not match issuerPermDiffOp.VsOperator
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerPermDiffOpID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "issuer permission vs_operator does not match operator",
		},
		{
			name: "Issuer authority mismatch",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority, // does not match issuerPermDiffAuth.Authority
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerPermDiffAuthID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "issuer participant authority does not match authority",
		},
		{
			name: "VS operator authz not enabled",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerPermNoAuthzID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "VS operator authorization is not enabled for permission",
		},
		{
			name: "Agent perm not found",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerPermID,
				VerifierParticipantId:    0,
				AgentParticipantId:       9999,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "agent permission not found",
		},
		{
			name: "Wallet agent perm not found",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerPermID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: 9999,
			},
			expectErr:  true,
			errMessage: "wallet agent permission not found",
		},
		{
			name: "Session update - authority mismatch",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              otherAuthority, // different from session creator
				Operator:                 operator,
				Id:                       sessionUUID, // same ID as first test case (already created)
				IssuerParticipantId:      issuerPermDiffAuthID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "session corporation does not match",
		},
		{
			name: "Valid update of existing session",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       sessionUUID, // same ID as first test case (already created)
				IssuerParticipantId:      issuerPermID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentPermID,
				WalletAgentParticipantId: walletAgentPermID,
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Configure delegation keeper error for this test case
			delKeeper.ErrToReturn = tc.setupErr
			defer func() { delKeeper.ErrToReturn = nil }()

			resp, err := ms.CreateOrUpdateParticipantSession(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.msg.Id, resp.Id)

				// Verify session was created/updated
				session, err := k.ParticipantSession.Get(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.AgentParticipantId, session.SessionRecords[len(session.SessionRecords)-1].AgentParticipantId)
				require.NotZero(t, session.CorporationId)
				require.Equal(t, tc.msg.Operator, session.VsOperator)

				// Check that the session contains an appropriate session record
				foundRecord := false
				for _, rec := range session.SessionRecords {
					if rec.IssuerParticipantId == tc.msg.IssuerParticipantId &&
						rec.VerifierParticipantId == tc.msg.VerifierParticipantId &&
						rec.WalletAgentParticipantId == tc.msg.WalletAgentParticipantId {
						foundRecord = true
						break
					}
				}
				require.True(t, foundRecord, "Session doesn't contain the expected session record")
			}
		})
	}
}

// TestDiscountApplicationInFeeCalculation tests that discounts are correctly applied when calculating fees
func TestDiscountApplicationInFeeCalculation(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, _ := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create validator perm (ISSUER_GRANTOR) with issuance fees
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		IssuanceFees:  100, // 100 trust units
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create ISSUER perm with discount set (per Issue #94: use discount instead of exemption)
	issuerPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		VsOperatorAuthzEnabled: true,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		IssuanceFeeDiscount:    5000, // 50% discount
		EffectiveFrom:          &pastTime,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Create agent perm
	agentPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	walletAgentPermID := agentPermID // Use same for simplicity

	t.Run("Discount applied to beneficiary fees", func(t *testing.T) {
		// When creating a session with issuerPermID:
		// 1. Sum fees from found_perm_set (validatorPerm with IssuanceFees=100)
		// 2. Apply exemption from issuerPerm: beneficiary_fees = 100 * (1 - 0.5) = 50
		// Expected: beneficiary_fees = 50

		msg := &types.MsgCreateOrUpdateParticipantSession{
			Corporation:              authority,
			Operator:                 operator,
			Id:                       uuid.New().String(),
			IssuerParticipantId:      issuerPermID,
			VerifierParticipantId:    0,
			AgentParticipantId:       agentPermID,
			WalletAgentParticipantId: walletAgentPermID,
		}

		resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, msg.Id, resp.Id)
	})

	t.Run("Discount applied in execution", func(t *testing.T) {
		// Create another issuer perm with different discount
		issuerPerm2 := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			VsOperator:             operator,
			VsOperatorAuthzEnabled: true,
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			IssuanceFeeDiscount:    3000, // 30% discount
			EffectiveFrom:          &pastTime,
		}
		issuerPerm2ID, err := k.CreatePermission(sdkCtx, issuerPerm2)
		require.NoError(t, err)

		// Expected: fees from validatorPerm (100) * (1 - 0.3) = 70
		msg := &types.MsgCreateOrUpdateParticipantSession{
			Corporation:              authority,
			Operator:                 operator,
			Id:                       uuid.New().String(),
			IssuerParticipantId:      issuerPerm2ID,
			VerifierParticipantId:    0,
			AgentParticipantId:       agentPermID,
			WalletAgentParticipantId: walletAgentPermID,
		}

		resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Multiple discounts applied", func(t *testing.T) {
		// Create validator with discount
		validatorWithDiscount := types.Participant{
			SchemaId:            1,
			Role:                types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:       trkKeeper.RegisterCorp(authority),
			Created:             &now,
			Adjusted:            &now,
			Modified:            &now,
			OpState:             types.OnboardingState_VALIDATED,
			IssuanceFees:        200,  // 200 trust units
			IssuanceFeeDiscount: 2000, // 20% discount
			EffectiveFrom:       &pastTime,
		}
		validatorWithDiscountID, err := k.CreatePermission(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		// Create issuer with discount (per Issue #94: use discount instead of exemption)
		issuerWithDiscount := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			VsOperator:             operator,
			VsOperatorAuthzEnabled: true,
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorWithDiscountID,
			OpState:                types.OnboardingState_VALIDATED,
			IssuanceFeeDiscount:    3000, // 30% discount
			EffectiveFrom:          &pastTime,
		}
		issuerWithDiscountID, err := k.CreatePermission(sdkCtx, issuerWithDiscount)
		require.NoError(t, err)

		require.NoError(t, err)

		// Expected calculation:
		// 1. Apply issuer discount: 200 * (1 - 0.3) = 140
		// Final beneficiary_fees = 140

		msg := &types.MsgCreateOrUpdateParticipantSession{
			Corporation:              authority,
			Operator:                 operator,
			Id:                       uuid.New().String(),
			IssuerParticipantId:      issuerWithDiscountID,
			VerifierParticipantId:    0,
			AgentParticipantId:       agentPermID,
			WalletAgentParticipantId: walletAgentPermID,
		}

		resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestGetParticipantByID tests the GetParticipantByID function
func TestGetParticipantByID(t *testing.T) {
	k, _, trkKeeper, _, ctx, _ := keepertest.PermissionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Create a test perm
	testPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}
	permID, err := k.CreatePermission(sdkCtx, testPerm)
	require.NoError(t, err)

	// Test getting the perm
	retrievedPerm, err := k.GetParticipantByID(sdkCtx, permID)
	require.NoError(t, err, "GetParticipantByID should not return an error for a valid ID")
	require.Equal(t, permID, retrievedPerm.Id, "Participant ID should match")
	require.Equal(t, testPerm.SchemaId, retrievedPerm.SchemaId, "Schema ID should match")
	require.Equal(t, testPerm.Role, retrievedPerm.Role, "Type should match")
	require.Equal(t, testPerm.CorporationId, retrievedPerm.CorporationId, "Corporation should match")

	// Test getting a non-existent perm
	_, err = k.GetParticipantByID(sdkCtx, 9999)
	require.Error(t, err, "GetParticipantByID should return an error for an invalid ID")
}

// TestCreateAndUpdatePermission tests the CreatePermission and UpdatePermission functions
func TestCreateAndUpdatePermission(t *testing.T) {
	k, _, trkKeeper, _, ctx, _ := keepertest.PermissionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Test CreatePermission
	testPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}

	permID, err := k.CreatePermission(sdkCtx, testPerm)
	require.NoError(t, err, "CreatePermission should not return an error")
	require.Greater(t, permID, uint64(0), "Participant ID should be greater than 0")

	// Retrieve the created perm
	retrievedPerm, err := k.GetParticipantByID(sdkCtx, permID)
	require.NoError(t, err)
	require.Equal(t, permID, retrievedPerm.Id, "Created perm ID should match")
	require.Equal(t, testPerm.SchemaId, retrievedPerm.SchemaId, "Created perm schema ID should match")

	// Test UpdatePermission
	futureTime := now.Add(24 * time.Hour)
	retrievedPerm.EffectiveUntil = &futureTime

	err = k.UpdatePermission(sdkCtx, retrievedPerm)
	require.NoError(t, err, "UpdatePermission should not return an error")

	// Retrieve the updated perm
	updatedPerm, err := k.GetParticipantByID(sdkCtx, permID)
	require.NoError(t, err)
	require.Equal(t, futureTime.Unix(), updatedPerm.EffectiveUntil.Unix(), "EffectiveUntil should be updated")
}

// TestQueryPermissions tests the query functions for permissions
func TestQueryPermissions(t *testing.T) {
	k, _, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// Create a trust registry
	trID := trkKeeper.CreateMockEcosystem(creator, validDid)

	// Create mock credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create several permissions for testing
	// Trust Registry perm
	trustPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		Did:           validDid,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	trustPermID, err := k.CreatePermission(sdkCtx, trustPerm)
	require.NoError(t, err)

	// Issuer perm
	issuerPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		Did:                    validDid,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Verifier perm
	verifierPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_VERIFIER,
		Did:                    validDid,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustPermID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	verifierPermID, err := k.CreatePermission(sdkCtx, verifierPerm)

	require.NoError(t, err)

	// Create a session for testing
	sessionID := uuid.New().String()
	session := types.ParticipantSession{
		Id:            sessionID,
		CorporationId: trkKeeper.RegisterCorp(creator),
		VsOperator:    creator,
		Created:       &now,
		Modified:      &now,
		SessionRecords: []*types.ParticipantSessionRecord{
			{
				Id:                    1,
				IssuerParticipantId:   issuerPermID,
				VerifierParticipantId: verifierPermID,
				AgentParticipantId:    issuerPermID, // Using issuer as agent for simplicity in test
			},
		},
	}
	err = k.ParticipantSession.Set(sdkCtx, sessionID, session)
	require.NoError(t, err)

	// Test GetParticipant query
	getPermReq := &types.QueryGetParticipantRequest{
		Id: issuerPermID,
	}
	getPermResp, err := k.GetParticipant(ctx, getPermReq)
	require.NoError(t, err)
	require.NotNil(t, getPermResp)
	require.Equal(t, issuerPermID, getPermResp.Participant.Id)
	require.Equal(t, validDid, getPermResp.Participant.Did)

	// Test ListParticipants query
	listPermReq := &types.QueryListParticipantsRequest{
		ResponseMaxSize: 10,
	}
	listPermResp, err := k.ListParticipants(ctx, listPermReq)
	require.NoError(t, err)
	require.NotNil(t, listPermResp)
	require.GreaterOrEqual(t, len(listPermResp.Participants), 3) // At least the 3 we created

	// Test GetParticipantSession query
	getSessionReq := &types.QueryGetParticipantSessionRequest{
		Id: sessionID,
	}
	getSessionResp, err := k.GetParticipantSession(ctx, getSessionReq)
	require.NoError(t, err)
	require.NotNil(t, getSessionResp)
	require.Equal(t, sessionID, getSessionResp.Session.Id)
	require.NotZero(t, getSessionResp.Session.CorporationId)

	// Test ListParticipantSessions query
	listSessionsReq := &types.QueryListParticipantSessionsRequest{
		ResponseMaxSize: 10,
	}
	listSessionsResp, err := k.ListParticipantSessions(ctx, listSessionsReq)
	require.NoError(t, err)
	require.NotNil(t, listSessionsResp)
	require.GreaterOrEqual(t, len(listSessionsResp.Sessions), 1) // At least the one we created

	// Test FindParticipantsWithDID query
	findPermDIDReq := &types.QueryFindParticipantsWithDIDRequest{
		Did:      validDid,
		Role:     uint32(types.ParticipantRole_ISSUER),
		SchemaId: 1,
	}
	findPermDIDResp, err := k.FindParticipantsWithDID(ctx, findPermDIDReq)
	require.NoError(t, err)
	require.NotNil(t, findPermDIDResp)
	require.Equal(t, 1, len(findPermDIDResp.Participants)) // Should find only the issuer perm
	require.Equal(t, issuerPermID, findPermDIDResp.Participants[0].Id)

	// Test FindBeneficiaries query
	findBenefReq := &types.QueryFindBeneficiariesRequest{
		IssuerParticipantId:   issuerPermID,
		VerifierParticipantId: verifierPermID,
	}
	findBenefResp, err := k.FindBeneficiaries(ctx, findBenefReq)
	require.NoError(t, err)
	require.NotNil(t, findBenefResp)
	require.GreaterOrEqual(t, len(findBenefResp.Participants), 1) // Should find the trust perm at minimum

	// Find the trust perm in the response
	foundTrustPerm := false
	for _, perm := range findBenefResp.Participants {
		if perm.Id == trustPermID {
			foundTrustPerm = true
			break
		}
	}
	require.True(t, foundTrustPerm, "Trust registry perm should be in beneficiaries")
}

func TestSlashPermissionTrustDeposit(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority__")).String()
	operator := sdk.AccAddress([]byte("test_operator___")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator__")).String()
	trControllerAddr := sdk.AccAddress([]byte("test_tr_ctrl____")).String()
	applicantAuthority := sdk.AccAddress([]byte("test_applicant__")).String()
	unauthorizedAddr := sdk.AccAddress([]byte("unauthorized_____")).String()

	// Create trust registry with trControllerAddr as controller
	validDid := "did:example:123456789abcdefghi"
	trID := trkKeeper.CreateMockEcosystem(trControllerAddr, validDid)

	// Create mock credential schema linked to the TR
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	// Create validator perm (ISSUER_GRANTOR) owned by validatorAddr
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create applicant perm (ISSUER) with deposit, vs_operator set
	applicantPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                1000,
		EffectiveFrom:          &pastTime,
		VsOperator:             operator,
		VsOperatorAuthzEnabled: true,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create a VERIFIER perm to test VS operator revocation
	verifierPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_VERIFIER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                500,
		EffectiveFrom:          &pastTime,
		VsOperator:             operator,
		VsOperatorAuthzEnabled: true,
	}
	verifierPermID, err := k.CreatePermission(sdkCtx, verifierPerm)
	require.NoError(t, err)

	// Create an ECOSYSTEM perm (no VS operator revocation for non-ISSUER/VERIFIER)
	ecosystemPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(applicantAuthority),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		Deposit:       300,
		EffectiveFrom: &pastTime,
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create expired perm (still slashable per spec)
	expiredTime := now.Add(-2 * time.Hour)
	expiredUntil := now.Add(-1 * time.Hour)
	expiredPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &expiredTime,
		Modified:               &expiredTime,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                200,
		EffectiveFrom:          &expiredTime,
		EffectiveUntil:         &expiredUntil,
	}
	expiredPermID, err := k.CreatePermission(sdkCtx, expiredPerm)
	require.NoError(t, err)

	// Create revoked perm (still slashable per spec)
	revokedPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                200,
		EffectiveFrom:          &pastTime,
		Revoked:                &now,
	}
	revokedPermID, err := k.CreatePermission(sdkCtx, revokedPerm)
	require.NoError(t, err)

	t.Run("AUTHZ check - operator authorization failure", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      100,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Nil(t, resp)
		delKeeper.ErrToReturn = nil // Reset
	})

	t.Run("Valid slash by validator ancestor", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      100,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.NotNil(t, perm.Slashed)
		require.Equal(t, uint64(100), perm.SlashedDeposit)
	})

	t.Run("Valid slash by TR controller", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: trControllerAddr,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      100,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(200), perm.SlashedDeposit) // cumulative: 100 + 100
	})

	t.Run("Valid slash on expired perm (still slashable per spec)", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          expiredPermID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, expiredPermID)
		require.NoError(t, err)
		require.NotNil(t, perm.Slashed)
		require.Equal(t, uint64(50), perm.SlashedDeposit)
	})

	t.Run("Valid slash on revoked perm (still slashable per spec)", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          revokedPermID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, revokedPermID)
		require.NoError(t, err)
		require.NotNil(t, perm.Slashed)
	})

	t.Run("VS operator revocation on VERIFIER perm", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          verifierPermID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("No VS operator revocation on ECOSYSTEM perm", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: trControllerAddr,
			Operator:    operator,
			Id:          ecosystemPermID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Participant not found", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          9999,
			Amount:      100,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission not found")
		require.Nil(t, resp)
	})

	t.Run("Amount exceeds deposit", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      999999,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "amount exceeds available deposit")
		require.Nil(t, resp)
	})

	t.Run("Unauthorized authority - not validator ancestor, not TR controller", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: unauthorizedAddr,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      10,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not authorized to slash this permission")
		require.Nil(t, resp)
	})

	t.Run("Wrong authority - applicant own authority cannot slash", func(t *testing.T) {
		// Unlike revoke, slash does NOT have Option #3 (self-authority)
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: applicantAuthority,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      10,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not authorized to slash this permission")
		require.Nil(t, resp)
	})

	_ = authority // suppress unused
}

func TestRepayPermissionSlashedTrustDeposit(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority_addr")).String()
	operator := sdk.AccAddress([]byte("test_operator_addr")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	otherAuthority := sdk.AccAddress([]byte("other_authority_ad")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	// Create ecosystem perm
	ecosystemPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create validator perm
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create applicant perm owned by authority with initial deposit
	applicantPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                1000,
		EffectiveFrom:          &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create unslashed perm (for negative test)
	unslashedPerm := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                500,
		EffectiveFrom:          &pastTime,
	}
	unslashedPermID, err := k.CreatePermission(sdkCtx, unslashedPerm)
	require.NoError(t, err)

	// Slash the applicant perm first
	slashMsg := &types.MsgSlashParticipantTrustDeposit{
		Corporation: validatorAddr,
		Operator:    validatorAddr,
		Id:          applicantPermID,
		Amount:      500,
	}
	_, err = ms.SlashParticipantTrustDeposit(ctx, slashMsg)
	require.NoError(t, err)

	// Verify slashed state
	slashedPerm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
	require.NoError(t, err)
	require.Equal(t, uint64(500), slashedPerm.SlashedDeposit)

	t.Run("AUTHZ check - operator authorization failure", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Nil(t, resp)
		delKeeper.ErrToReturn = nil
	})

	t.Run("Valid repayment by owner authority", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          applicantPermID,
			Amount:      500,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify perm was updated correctly
		perm, err := k.GetParticipantByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.NotNil(t, perm.Repaid)
		require.NotNil(t, perm.Modified)
		require.Equal(t, uint64(500), perm.RepaidDeposit)
	})

	t.Run("Invalid - already fully repaid", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "slashed deposit already fully repaid")
		require.Nil(t, resp)
	})

	t.Run("Invalid - perm not found", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          9999,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm not found")
		require.Nil(t, resp)
	})

	t.Run("Invalid - wrong authority (not owner)", func(t *testing.T) {
		// Slash a new perm for this test
		newPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(otherAuthority),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorPermID,
			OpState:                types.OnboardingState_VALIDATED,
			Deposit:                300,
			EffectiveFrom:          &pastTime,
		}
		otherPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)
		// Slash it
		_, err = ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    validatorAddr,
			Id:          otherPermID,
			Amount:      100,
		})
		require.NoError(t, err)

		// Try to repay with wrong authority
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority, // wrong - perm belongs to otherAuthority
			Operator:    operator,
			Id:          otherPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not the owner of this participant")
		require.Nil(t, resp)
	})

	t.Run("Invalid - no slashed deposit to repay", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          unslashedPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no slashed timestamp")
		require.Nil(t, resp)
	})
}

func TestCreatePermission(t *testing.T) {
	k, ms, mockCsKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority_addr")).String()
	operator := sdk.AccAddress([]byte("test_operator_addr")).String()
	otherAuthority := sdk.AccAddress([]byte("other_authority_ad")).String()
	validDid := "did:example:123456789abcdefghi"
	now := sdkCtx.BlockTime()

	trID := trkKeeper.CreateMockEcosystem(authority, validDid)
	mockCsKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)

	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(24 * time.Hour)
	farFuture := now.Add(360 * 24 * time.Hour)

	// Create ecosystem perm (active, with effective_until)
	ecosystemPerm := types.Participant{
		SchemaId:       1,
		Role:           types.ParticipantRole_ECOSYSTEM,
		Did:            validDid,
		CorporationId:  trkKeeper.RegisterCorp(authority),
		Created:        &now,
		Modified:       &now,
		OpState:        types.OnboardingState_VALIDATED,
		EffectiveFrom:  &pastTime,
		EffectiveUntil: &farFuture,
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create ecosystem perm without effective_until (never expires)
	neverExpirePerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		Did:           validDid,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	neverExpirePermID, err := k.CreatePermission(sdkCtx, neverExpirePerm)
	require.NoError(t, err)

	t.Run("AUTHZ check - operator authorization failure", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Nil(t, resp)
		delKeeper.ErrToReturn = nil
	})

	t.Run("Valid ISSUER permission", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
			VerificationFees:       100,
			ValidationFees:         50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_ISSUER, perm.Role)
		require.NotZero(t, perm.CorporationId)
		require.Equal(t, validDid, perm.Did)
		require.Equal(t, ecosystemPermID, perm.ValidatorParticipantId)
		require.Equal(t, uint64(1), perm.SchemaId) // inherited from validator_perm
		require.Equal(t, uint64(100), perm.VerificationFees)
		require.Equal(t, uint64(50), perm.ValidationFees)
		require.Equal(t, uint64(0), perm.IssuanceFees)
		require.Equal(t, uint64(0), perm.Deposit)
		require.NotNil(t, perm.Created)
		require.NotNil(t, perm.Modified)
	})

	t.Run("Valid VERIFIER permission", func(t *testing.T) {
		futureTime2 := futureTime.Add(1 * time.Hour)
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    "did:example:verifier1",
			EffectiveFrom:          &futureTime2,
			EffectiveUntil:         &farFuture,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_VERIFIER, perm.Role)
		require.Equal(t, uint64(0), perm.VerificationFees)
		require.Equal(t, uint64(0), perm.ValidationFees)
	})

	t.Run("Invalid - validator perm not found", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: 9999,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator permission not found")
		require.Nil(t, resp)
	})

	t.Run("Invalid - validator perm not ECOSYSTEM", func(t *testing.T) {
		// Create a non-ecosystem perm
		issuerPerm := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: ecosystemPermID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: issuerPermID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "ECOSYSTEM permission")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_from not in future", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    validDid,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_from must be in the future")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_until before effective_from", func(t *testing.T) {
		beforeFuture := futureTime.Add(-1 * time.Minute)
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &beforeFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be greater than effective_from")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_until exceeds validator_perm", func(t *testing.T) {
		wayFuture := farFuture.Add(24 * time.Hour)
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &wayFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be <= validator_perm.effective_until")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_until null but validator_perm has effective_until", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			// EffectiveUntil nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be set when validator_perm has effective_until")
		require.Nil(t, resp)
	})

	t.Run("Valid - both effective_until null when validator_perm never expires", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            otherAuthority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: neverExpirePermID,
			Did:                    "did:example:neverexpire",
			EffectiveFrom:          &futureTime,
			// EffectiveUntil nil - OK because validator_perm also has nil
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Invalid - VERIFIER with validation_fees", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: ecosystemPermID,
			Did:                    "did:example:verifier2",
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
			ValidationFees:         100,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation_fees")
		require.Nil(t, resp)
	})

	t.Run("Invalid - non-OPEN management mode", func(t *testing.T) {
		mockCsKeeper.UpdateMockCredentialSchema(2, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		// Create ecosystem perm for schema 2
		ecoPermS2 := types.Participant{
			SchemaId:       2,
			Role:           types.ParticipantRole_ECOSYSTEM,
			CorporationId:  trkKeeper.RegisterCorp(authority),
			Created:        &now,
			Modified:       &now,
			OpState:        types.OnboardingState_VALIDATED,
			EffectiveFrom:  &pastTime,
			EffectiveUntil: &farFuture,
		}
		ecoPermS2ID, err := k.CreatePermission(sdkCtx, ecoPermS2)
		require.NoError(t, err)

		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecoPermS2ID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not OPEN")
		require.Nil(t, resp)
	})
}

// =============================================================================
// ISSUE #191: CreateRootParticipant - effective_from MUST be set
// =============================================================================
// This test validates that CreateRootParticipant requires effective_from to be set
// and it must be in the future. Per spec [MOD-PERM-MSG-7-2-1]:
// - effective_from is mandatory
// - effective_from must be in the future

func TestCreateRootPermission(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validDid := "did:example:123456789abcdefghi"
	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := authority // self-delegation
	otherAddr := sdk.AccAddress([]byte("other_address_______")).String()

	// Create trust registry where authority is the controller
	trID := trkKeeper.CreateMockEcosystem(authority, validDid)

	// Create credential schema linked to the trust registry
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	futureTime := now.Add(1 * time.Hour)
	pastTime := now.Add(-1 * time.Hour)
	farFutureTime := now.Add(24 * time.Hour)
	veryFarFuture := now.Add(48 * time.Hour)

	testCases := []struct {
		name      string
		msg       *types.MsgCreateRootParticipant
		expectErr bool
		errMsg    string
	}{
		// === Basic checks [MOD-PERM-MSG-7-2-1] ===
		{
			name: "1. Reject nil effective_from",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 1, Did: validDid,

				EffectiveFrom: nil,
			},
			expectErr: true,
			errMsg:    "effective_from is required",
		},
		{
			name: "2. Reject past effective_from",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 1, Did: validDid,

				EffectiveFrom: &pastTime,
			},
			expectErr: true,
			errMsg:    "effective_from must be in the future",
		},
		{
			name: "3. Reject effective_from equal to now",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 1, Did: validDid,

				EffectiveFrom: &now,
			},
			expectErr: true,
			errMsg:    "effective_from must be in the future",
		},
		{
			name: "4. Reject effective_until <= effective_from",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 1, Did: validDid,

				EffectiveFrom:  &futureTime,
				EffectiveUntil: &futureTime, // equal, not greater
			},
			expectErr: true,
			errMsg:    "effective_until must be greater than effective_from",
		},
		{
			name: "5. Reject invalid schema ID (not found)",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 999, Did: validDid,
				EffectiveFrom: &futureTime,
			},
			expectErr: true,
			errMsg:    "credential schema not found",
		},
		// === Participant checks [MOD-PERM-MSG-7-2-2] ===
		{
			name: "6. Reject authority not TR controller",
			msg: &types.MsgCreateRootParticipant{
				Corporation: otherAddr, Operator: otherAddr,
				SchemaId: 1, Did: validDid,

				EffectiveFrom:  &futureTime,
				EffectiveUntil: &farFutureTime,
			},
			expectErr: true,
			errMsg:    "does not control",
		},
		// === Happy path [MOD-PERM-MSG-7-3] ===
		{
			name: "7. Happy path with effective_until",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 1, Did: validDid,
				EffectiveFrom:    &futureTime,
				EffectiveUntil:   &farFutureTime,
				ValidationFees:   100,
				IssuanceFees:     200,
				VerificationFees: 300,
			},
			expectErr: false,
		},
		{
			name: "8. Happy path with nil effective_until (never expires)",
			msg: &types.MsgCreateRootParticipant{
				Corporation: authority, Operator: operator,
				SchemaId: 1, Did: "did:example:second",
				EffectiveFrom:  &veryFarFuture,
				EffectiveUntil: nil,
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CreateRootParticipant(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// [MOD-PERM-MSG-7-3] verify created permission per spec v4 draft 13:
				// perm.type is hardcoded to ECOSYSTEM, and perm.vs_operator is not set by this message.
				perm, err := k.GetParticipantByID(sdkCtx, resp.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.SchemaId, perm.SchemaId)
				require.Equal(t, types.ParticipantRole_ECOSYSTEM, perm.Role,
					"Create Root Participant MUST hardcode perm.type to ECOSYSTEM per spec [MOD-PERM-MSG-7-3]")
				require.Empty(t, perm.VsOperator,
					"Create Root Participant MUST NOT set perm.vs_operator per spec [MOD-PERM-MSG-7-3]")
				require.Equal(t, tc.msg.Did, perm.Did)
				require.NotZero(t, perm.CorporationId)
				require.Equal(t, now, *perm.Created)
				require.Equal(t, now, *perm.Modified)
				require.Equal(t, tc.msg.EffectiveFrom.Unix(), perm.EffectiveFrom.Unix())
				if tc.msg.EffectiveUntil != nil {
					require.Equal(t, tc.msg.EffectiveUntil.Unix(), perm.EffectiveUntil.Unix())
				} else {
					require.Nil(t, perm.EffectiveUntil)
				}
				require.Equal(t, tc.msg.ValidationFees, perm.ValidationFees)
				require.Equal(t, tc.msg.IssuanceFees, perm.IssuanceFees)
				require.Equal(t, tc.msg.VerificationFees, perm.VerificationFees)
				require.Equal(t, uint64(0), perm.Deposit)
			}
		})
	}
}

func TestCreateRootPermission_OverlapChecks(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validDid := "did:example:overlap_test"
	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := authority

	trID := trkKeeper.CreateMockEcosystem(authority, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()

	// Create an existing permission: effective_from=+1h, effective_until=+24h
	existingFrom := now.Add(1 * time.Hour)
	existingUntil := now.Add(24 * time.Hour)
	resp, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
		Corporation: authority, Operator: operator,
		SchemaId: 1, Did: validDid,

		EffectiveFrom:  &existingFrom,
		EffectiveUntil: &existingUntil,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	t.Run("1. Overlap: new effective_from before existing effective_until", func(t *testing.T) {
		// new effective_from = +12h, existing effective_until = +24h
		// existing.effective_until > new.effective_from → abort
		newFrom := now.Add(12 * time.Hour)
		newUntil := now.Add(48 * time.Hour)
		_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 1, Did: validDid,

			EffectiveFrom:  &newFrom,
			EffectiveUntil: &newUntil,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap")
	})

	t.Run("2. Overlap: existing effective_from before new effective_until", func(t *testing.T) {
		// new effective_from = +25h (after existing), new effective_until = +48h
		// But existing.effective_from (+1h) < new.effective_until (+48h) → abort
		newFrom := now.Add(25 * time.Hour)
		newUntil := now.Add(48 * time.Hour)
		_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 1, Did: validDid,

			EffectiveFrom:  &newFrom,
			EffectiveUntil: &newUntil,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap")
	})

	t.Run("3. Overlap: existing perm with nil effective_until (never expires)", func(t *testing.T) {
		// Create a new schema to test with nil effective_until
		csKeeper.UpdateMockCredentialSchema(2, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		neverExpiresFrom := now.Add(1 * time.Hour)
		resp2, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 2, Did: validDid,

			EffectiveFrom:  &neverExpiresFrom,
			EffectiveUntil: nil, // Never expires
		})
		require.NoError(t, err)
		require.NotNil(t, resp2)

		// Now try to create another one → should fail because existing never expires
		newFrom := now.Add(48 * time.Hour)
		newUntil := now.Add(72 * time.Hour)
		_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 2, Did: validDid,

			EffectiveFrom:  &newFrom,
			EffectiveUntil: &newUntil,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "never expires")
	})

	t.Run("4. Revoked/slashed/repaid perms excluded from overlap", func(t *testing.T) {
		// Create a new schema to test with
		csKeeper.UpdateMockCredentialSchema(3, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		revokedFrom := now.Add(1 * time.Hour)
		revokedUntil := now.Add(100 * time.Hour)
		resp3, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 3, Did: validDid,

			EffectiveFrom:  &revokedFrom,
			EffectiveUntil: &revokedUntil,
		})
		require.NoError(t, err)

		// Mark the perm as revoked
		perm, err := k.GetParticipantByID(sdkCtx, resp3.Id)
		require.NoError(t, err)
		revokedTime := now
		perm.Revoked = &revokedTime
		err = k.Participant.Set(sdkCtx, perm.Id, perm)
		require.NoError(t, err)

		// Now create a new perm that would overlap if the revoked one was active → should succeed
		newFrom := now.Add(2 * time.Hour)
		newUntil := now.Add(50 * time.Hour)
		_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 3, Did: validDid,

			EffectiveFrom:  &newFrom,
			EffectiveUntil: &newUntil,
		})
		require.NoError(t, err)
	})

	t.Run("5. No overlap: new perm starts after existing ends", func(t *testing.T) {
		// Use schema 1 with existing perm: +1h to +24h
		// But existing.effective_from < new.effective_until still causes overlap
		// To truly avoid overlap, need perm on a different schema OR existing must be expired/revoked
		csKeeper.UpdateMockCredentialSchema(4, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		firstFrom := now.Add(1 * time.Hour)
		firstUntil := now.Add(5 * time.Hour)
		_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 4, Did: validDid,

			EffectiveFrom:  &firstFrom,
			EffectiveUntil: &firstUntil,
		})
		require.NoError(t, err)

		// New perm starts after first ends: +6h to +10h
		// existing.effective_until (+5h) < new.effective_from (+6h) → OK
		// existing.effective_from (+1h) < new.effective_until (+10h) → overlap!
		// Per spec this is still an overlap, so it should fail
		secondFrom := now.Add(6 * time.Hour)
		secondUntil := now.Add(10 * time.Hour)
		_, err = ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 4, Did: validDid,

			EffectiveFrom:  &secondFrom,
			EffectiveUntil: &secondUntil,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap")
	})
}

func TestCreateRootPermission_AuthzCheck(t *testing.T) {
	_, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validDid := "did:example:authzcheck"
	authority := sdk.AccAddress([]byte("test_authority______")).String()
	operator := authority

	trID := trkKeeper.CreateMockEcosystem(authority, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	futureTime := sdkCtx.BlockTime().Add(1 * time.Hour)
	farFuture := sdkCtx.BlockTime().Add(24 * time.Hour)

	t.Run("AUTHZ-CHECK failure aborts", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		defer func() { delKeeper.ErrToReturn = nil }()

		_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 1, Did: validDid,

			EffectiveFrom:  &futureTime,
			EffectiveUntil: &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
	})

	t.Run("AUTHZ-CHECK success allows creation", func(t *testing.T) {
		resp, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{
			Corporation: authority, Operator: operator,
			SchemaId: 1, Did: validDid,

			EffectiveFrom:  &futureTime,
			EffectiveUntil: &farFuture,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// =============================================================================
// ISSUE #193: StartParticipantOP - Validator permission must be ACTIVE
// =============================================================================
// This test validates that StartParticipantOP requires the validator permission
// to be ACTIVE (not INACTIVE, REVOKED, EXPIRED, etc). Per spec:
// - validator_perm must be a valid permission
// - If effective_from is null or in the future, perm is INACTIVE/FUTURE
// - If revoked, slashed, or expired, perm is invalid

func TestStartPermissionVP_ValidatorMustBeActive(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// Create trust registry
	trID := trkKeeper.CreateMockEcosystem(creator, validDid)

	// Create mock credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)     // In the past - for ACTIVE permissions
	futureTime := now.Add(1 * time.Hour)    // In the future - for FUTURE/INACTIVE permissions
	expiredTime := now.Add(-24 * time.Hour) // Far in the past - for EXPIRED permissions

	// Create an ACTIVE validator permission (valid case for comparison)
	activeValidatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // In the past = ACTIVE
	}
	activeValidatorPermID, err := k.CreatePermission(sdkCtx, activeValidatorPerm)
	require.NoError(t, err)

	// Issue #193: Create a validator permission with NO effective_from (INACTIVE)
	inactiveValidatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: nil, // NULL effective_from = INACTIVE
	}
	inactiveValidatorPermID, err := k.CreatePermission(sdkCtx, inactiveValidatorPerm)
	require.NoError(t, err)

	// Issue #193: Create a validator permission with FUTURE effective_from
	futureValidatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &futureTime, // Future effective_from = not yet ACTIVE
	}
	futureValidatorPermID, err := k.CreatePermission(sdkCtx, futureValidatorPerm)
	require.NoError(t, err)

	// Issue #193: Create an EXPIRED validator permission
	expiredValidatorPerm := types.Participant{
		SchemaId:       1,
		Role:           types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId:  trkKeeper.RegisterCorp(creator),
		Created:        &now,
		Adjusted:       &now,
		Modified:       &now,
		OpState:        types.OnboardingState_VALIDATED,
		EffectiveFrom:  &expiredTime,
		EffectiveUntil: &pastTime, // Already expired
	}
	expiredValidatorPermID, err := k.CreatePermission(sdkCtx, expiredValidatorPerm)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		msg       *types.MsgStartParticipantOP
		expectErr bool
		errMsg    string
	}{
		{
			// Baseline: Active validator should work
			name: "Issue #193: Accept ACTIVE validator - valid case",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: activeValidatorPermID,
				Did:                    validDid,
			},
			expectErr: false,
			errMsg:    "",
		},
		{
			// Issue #193: Validator with null effective_from should be rejected
			name: "Issue #193: Reject INACTIVE validator - effective_from is null",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: inactiveValidatorPermID,
				Did:                    validDid,
			},
			expectErr: true,
			errMsg:    "validator perm is not valid",
		},
		{
			// Issue #193: Validator with future effective_from should be rejected
			name: "Issue #193: Reject FUTURE validator - effective_from is in the future",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: futureValidatorPermID,
				Did:                    validDid,
			},
			expectErr: true,
			errMsg:    "validator perm is not valid",
		},
		{
			// Issue #193: Expired validator should be rejected
			name: "Issue #193: Reject EXPIRED validator - effective_until has passed",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: expiredValidatorPermID,
				Did:                    validDid,
			},
			expectErr: true,
			errMsg:    "validator perm is not valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.StartParticipantOP(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}

// =============================================================================
// ISSUE #196: RevokeParticipant - Allow revoking not-yet-active permissions
// =============================================================================
// This test validates that RevokeParticipant allows revoking permissions that
// are not yet active (e.g., effective_from is in the future or null).
// Per spec, no IsValidPermission check is required for revocation.

// TestRevokePermission_RequiresActivePermission tests that v4 spec requires
// applicant_perm to be an active permission (reverting Issue #196 relaxation).
func TestRevokePermission_RequiresActivePermission(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority__")).String()
	operatorAddr := sdk.AccAddress([]byte("test_operator___")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	// Create an ACTIVE permission (for comparison)
	activePerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // ACTIVE
	}
	activePermID, err := k.CreatePermission(sdkCtx, activePerm)
	require.NoError(t, err)

	// Create a permission with FUTURE effective_from (not yet active)
	futurePerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &futureTime, // FUTURE - not yet active
	}
	futurePermID, err := k.CreatePermission(sdkCtx, futurePerm)
	require.NoError(t, err)

	// Create a permission with NULL effective_from (inactive)
	inactivePerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: nil, // INACTIVE - no effective_from
	}
	inactivePermID, err := k.CreatePermission(sdkCtx, inactivePerm)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		msg       *types.MsgRevokeParticipant
		expectErr bool
		errMsg    string
	}{
		{
			// Baseline: Revoking an ACTIVE permission should work
			name: "Revoke ACTIVE permission - valid case",
			msg: &types.MsgRevokeParticipant{
				Corporation: authority,
				Operator:    operatorAddr,
				Id:          activePermID,
			},
			expectErr: false,
			errMsg:    "",
		},
		{
			// v4 spec: FUTURE permission (not yet active) should be rejected
			name: "Revoke FUTURE permission - not yet active should be rejected",
			msg: &types.MsgRevokeParticipant{
				Corporation: authority,
				Operator:    operatorAddr,
				Id:          futurePermID,
			},
			expectErr: true,
			errMsg:    "applicant permission is not active",
		},
		{
			// v4 spec: INACTIVE permission (null effective_from) should be rejected
			name: "Revoke INACTIVE permission - null effective_from should be rejected",
			msg: &types.MsgRevokeParticipant{
				Corporation: authority,
				Operator:    operatorAddr,
				Id:          inactivePermID,
			},
			expectErr: true,
			errMsg:    "applicant permission is not active",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RevokeParticipant(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify the permission was revoked
				perm, err := k.GetParticipantByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, perm.Revoked, "Participant should be revoked")
			}
		})
	}
}

// TestStartPermissionVP_OverlapCheck tests [MOD-PERM-MSG-1-2-4]:
// Cannot have 2 active VPs in the same (schema_id, type, validator_participant_id, authority) context.
func TestStartPermissionVP_OverlapCheck(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := trkKeeper.CreateMockEcosystem(creator, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// First VP should succeed
	msg := &types.MsgStartParticipantOP{
		Corporation:            creator,
		Operator:               creator,
		Role:                   types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorPermID,
		Did:                    validDid,
	}
	resp, err := ms.StartParticipantOP(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Second VP with same (schema_id, type, validator_participant_id, authority) should fail
	t.Run("Duplicate PENDING VP in same context", func(t *testing.T) {
		msg2 := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               creator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorPermID,
			Did:                    "did:example:different-did",
		}
		resp2, err := ms.StartParticipantOP(ctx, msg2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap check failed")
		require.Contains(t, err.Error(), "an active validation process already exists")
		require.Nil(t, resp2)
	})

	// Different authority should succeed (no overlap)
	t.Run("Different authority no overlap", func(t *testing.T) {
		otherCreator := sdk.AccAddress([]byte("other_creator")).String()
		msg3 := &types.MsgStartParticipantOP{
			Corporation:            otherCreator,
			Operator:               otherCreator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorPermID,
			Did:                    validDid,
		}
		resp3, err := ms.StartParticipantOP(ctx, msg3)
		require.NoError(t, err)
		require.NotNil(t, resp3)
	})

	// Different type should succeed (no overlap)
	t.Run("Different type no overlap", func(t *testing.T) {
		// Need a VERIFIER_GRANTOR validator for VERIFIER type
		csKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		verifierGrantorPerm := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgPermID, err := k.CreatePermission(sdkCtx, verifierGrantorPerm)
		require.NoError(t, err)

		msg4 := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               creator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: vgPermID,
			Did:                    validDid,
		}
		resp4, err := ms.StartParticipantOP(ctx, msg4)
		require.NoError(t, err)
		require.NotNil(t, resp4)
	})
}

// TestStartPermissionVP_AuthzCheck tests that the AUTHZ-CHECK via DelegationKeeper
// is properly enforced when the keeper is present.
func TestStartPermissionVP_AuthzCheck(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := trkKeeper.CreateMockEcosystem(creator, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("AUTHZ-CHECK failure blocks StartParticipantOP", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator not authorized for authority")
		defer func() { delKeeper.ErrToReturn = nil }()

		msg := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               sdk.AccAddress([]byte("unauthorized_op")).String(),
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: 1,
			Did:                    validDid,
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Contains(t, err.Error(), "operator not authorized")
		require.Nil(t, resp)
	})

	t.Run("AUTHZ-CHECK success allows StartParticipantOP", func(t *testing.T) {
		delKeeper.ErrToReturn = nil

		msg := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               creator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: 1,
			Did:                    validDid,
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestStartPermissionVP_VsOperatorAndFields tests that vs_operator fields and DID are correctly
// persisted, and that empty DID is rejected at the keeper level.
func TestStartPermissionVP_VsOperatorAndFields(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	vsOperator := sdk.AccAddress([]byte("vs_operator_acct")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := trkKeeper.CreateMockEcosystem(creator, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	validatorPerm := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("vs_operator fields propagated to stored permission", func(t *testing.T) {
		operator := sdk.AccAddress([]byte("diff_operator_aa")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorPermID,
			Did:                    validDid,
			VsOperator:             vsOperator,
			VsOperatorAuthzEnabled: true,
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, validDid, perm.Did, "DID should be stored")
		require.NotZero(t, perm.CorporationId, "Corporation should be set")
		require.Equal(t, vsOperator, perm.VsOperator, "VsOperator should be stored")
		require.True(t, perm.VsOperatorAuthzEnabled, "VsOperatorAuthzEnabled should be true")
		require.Equal(t, uint64(1), perm.SchemaId, "SchemaId should be derived from validator perm")
		require.Equal(t, types.OnboardingState_PENDING, perm.OpState)
	})

	t.Run("VERIFIER with VERIFIER_GRANTOR validator", func(t *testing.T) {
		vgPerm := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgPermID, err := k.CreatePermission(sdkCtx, vgPerm)
		require.NoError(t, err)

		verifierCreator := sdk.AccAddress([]byte("verifier_creator")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            verifierCreator,
			Operator:               verifierCreator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: vgPermID,
			Did:                    "did:example:verifier-did-123",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_VERIFIER, perm.Role)
		require.Equal(t, vgPermID, perm.ValidatorParticipantId)
	})

	t.Run("HOLDER with ISSUER validator", func(t *testing.T) {
		// Create ISSUER perm to serve as validator for HOLDER
		issuerPerm := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_ISSUER,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		holderCreator := sdk.AccAddress([]byte("holder_creator_a")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            holderCreator,
			Operator:               holderCreator,
			Role:                   types.ParticipantRole_HOLDER,
			ValidatorParticipantId: issuerPermID,
			Did:                    "did:example:holder-did-456",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_HOLDER, perm.Role)
		require.Equal(t, issuerPermID, perm.ValidatorParticipantId)
	})

	t.Run("HOLDER with wrong validator type rejects", func(t *testing.T) {
		holderCreator := sdk.AccAddress([]byte("holder_bad_val_a")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            holderCreator,
			Operator:               holderCreator,
			Role:                   types.ParticipantRole_HOLDER,
			ValidatorParticipantId: validatorPermID, // ISSUER_GRANTOR, not ISSUER
			Did:                    "did:example:holder-bad-val",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "holder perm requires ISSUER validator")
		require.Nil(t, resp)
	})

	t.Run("ECOSYSTEM type combination - ISSUER_GRANTOR with ECOSYSTEM validator", func(t *testing.T) {
		// Create schema with ECOSYSTEM mode for issuer
		csKeeper.UpdateMockCredentialSchema(2, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		ecosystemPerm := types.Participant{
			SchemaId:      2,
			Role:          types.ParticipantRole_ECOSYSTEM,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		ecoPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
		require.NoError(t, err)

		grantorCreator := sdk.AccAddress([]byte("grantor_eco_crea")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            grantorCreator,
			Operator:               grantorCreator,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			ValidatorParticipantId: ecoPermID,
			Did:                    "did:example:issuer-grantor-eco",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_ISSUER_GRANTOR, perm.Role)
	})
}

// =============================================================================
// VSOA Grant / Revoke Tests (MSG-5 / MSG-6)
// =============================================================================

// TestVSOA_GrantWithFeegrant verifies that when SetParticipantOPToValidated is
// called for an ISSUER permission with VsOperatorAuthzEnabled=true and
// VsOperatorAuthzWithFeegrant=true, both AddPermToVSOA and GrantFeeAllowance
// are invoked on the delegation keeper.
func TestVSOA_GrantWithFeegrant(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validatorAddr := sdk.AccAddress([]byte("test_validator______")).String()
	applicantAddr := sdk.AccAddress([]byte("test_applicant______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	// Create active ISSUER_GRANTOR validator perm
	validatorPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	// Create PENDING ISSUER perm with VSOA fields
	applicantPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(applicantAddr),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		ValidatorParticipantId:      validatorPermID,
		OpState:                     types.OnboardingState_PENDING,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Configure mock: GetVSOAPermissions returns the perm we just added
	delKeeper.GetVSOAPermissionsResult = []uint64{applicantPermID}

	delKeeper.Reset()

	resp, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         validatorAddr,
		Id:               applicantPermID,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveUntil:   &futureTime,
		OpSummaryDigest:  "sha384-validDigest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify AddPermToVSOA was called
	require.Len(t, delKeeper.AddPermToVSOACalls, 1)
	require.Equal(t, applicantAddr, delKeeper.AddPermToVSOACalls[0].Authority)
	require.Equal(t, vsOperator, delKeeper.AddPermToVSOACalls[0].VsOperator)
	require.Equal(t, applicantPermID, delKeeper.AddPermToVSOACalls[0].PermID)

	// Verify GrantFeeAllowance was called (feegrant enabled)
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 1)
	require.Equal(t, applicantAddr, delKeeper.GrantFeeAllowanceCalls[0].Authority)
	require.Equal(t, vsOperator, delKeeper.GrantFeeAllowanceCalls[0].Grantee)
}

// TestVSOA_GrantWithoutFeegrant verifies that when VsOperatorAuthzEnabled=true
// but VsOperatorAuthzWithFeegrant=false, AddPermToVSOA is called but
// GrantFeeAllowance is NOT called.
func TestVSOA_GrantWithoutFeegrant(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validatorAddr := sdk.AccAddress([]byte("test_validator______")).String()
	applicantAddr := sdk.AccAddress([]byte("test_applicant______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	validatorPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	applicantPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(applicantAddr),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		ValidatorParticipantId:      validatorPermID,
		OpState:                     types.OnboardingState_PENDING,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: false, // feegrant disabled
	})
	require.NoError(t, err)

	delKeeper.Reset()

	resp, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         validatorAddr,
		Id:               applicantPermID,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveUntil:   &futureTime,
		OpSummaryDigest:  "sha384-validDigest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// AddPermToVSOA should be called
	require.Len(t, delKeeper.AddPermToVSOACalls, 1)

	// GrantFeeAllowance should NOT be called
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 0)
}

// TestVSOA_GrantSkipsWhenVsOperatorEmpty verifies early return when
// the permission has no VsOperator set.
func TestVSOA_GrantSkipsWhenVsOperatorEmpty(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validatorAddr := sdk.AccAddress([]byte("test_validator______")).String()
	applicantAddr := sdk.AccAddress([]byte("test_applicant______")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	validatorPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	applicantPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(applicantAddr),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		ValidatorParticipantId:      validatorPermID,
		OpState:                     types.OnboardingState_PENDING,
		VsOperator:                  "", // empty — should skip
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	delKeeper.Reset()

	resp, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         validatorAddr,
		Id:               applicantPermID,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveUntil:   &futureTime,
		OpSummaryDigest:  "sha384-validDigest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Neither AddPermToVSOA nor GrantFeeAllowance should be called
	require.Len(t, delKeeper.AddPermToVSOACalls, 0)
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 0)
}

// TestVSOA_GrantSkipsWhenNotEnabled verifies early return when
// VsOperatorAuthzEnabled is false (even if VsOperator is set).
func TestVSOA_GrantSkipsWhenNotEnabled(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validatorAddr := sdk.AccAddress([]byte("test_validator______")).String()
	applicantAddr := sdk.AccAddress([]byte("test_applicant______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	validatorPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	})
	require.NoError(t, err)

	applicantPermID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(applicantAddr),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		ValidatorParticipantId:      validatorPermID,
		OpState:                     types.OnboardingState_PENDING,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      false, // not enabled
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	delKeeper.Reset()

	resp, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         validatorAddr,
		Id:               applicantPermID,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveUntil:   &futureTime,
		OpSummaryDigest:  "sha384-validDigest",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Grant path should be skipped entirely
	require.Len(t, delKeeper.AddPermToVSOACalls, 0)
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 0)
}

// TestVSOA_RevokeRemovesAndRevokesFeegrantWhenLastPerm verifies that when
// RevokeParticipant is called on the last perm in a VSOA, RemovePermFromVSOA
// is called and RevokeFeeAllowance is called (feegrant fully revoked).
func TestVSOA_RevokeRemovesAndRevokesFeegrantWhenLastPerm(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	// Create an active ISSUER perm with VSOA fields
	permID, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Configure mock: RemovePermFromVSOA returns empty slice (no remaining perms)
	delKeeper.RemovePermFromVSOARemainingPerms = []uint64{}

	delKeeper.Reset()

	_, err = ms.RevokeParticipant(ctx, &types.MsgRevokeParticipant{
		Corporation: authority,
		Operator:    authority,
		Id:          permID,
	})
	require.NoError(t, err)

	// Verify RemovePermFromVSOA was called
	require.Len(t, delKeeper.RemovePermFromVSOACalls, 1)
	require.Equal(t, authority, delKeeper.RemovePermFromVSOACalls[0].Authority)
	require.Equal(t, vsOperator, delKeeper.RemovePermFromVSOACalls[0].VsOperator)
	require.Equal(t, permID, delKeeper.RemovePermFromVSOACalls[0].PermID)

	// Verify RevokeFeeAllowance was called (last perm, so full revoke)
	require.Len(t, delKeeper.RevokeFeeAllowanceCalls, 1)
	require.Equal(t, authority, delKeeper.RevokeFeeAllowanceCalls[0].Authority)
	require.Equal(t, vsOperator, delKeeper.RevokeFeeAllowanceCalls[0].Grantee)

	// GrantFeeAllowance should NOT be called (we revoke, not recalculate)
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 0)
}

// TestVSOA_RevokeRecalculatesFeegrantWhenOtherPermsRemain verifies that when
// RevokeParticipant is called but other perms remain in the VSOA, the feegrant
// is recalculated (GrantFeeAllowance called with new expiry) rather than revoked.
func TestVSOA_RevokeRecalculatesFeegrantWhenOtherPermsRemain(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime1 := now.Add(365 * 24 * time.Hour)
	futureTime2 := now.Add(730 * 24 * time.Hour) // 2 years out

	// Create active ISSUER perm #1 (the one we will revoke)
	permID1, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime1,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Create active ISSUER perm #2 (remains in VSOA after revoke)
	permID2, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime2,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Configure mock: RemovePermFromVSOA returns remaining perm IDs
	delKeeper.RemovePermFromVSOARemainingPerms = []uint64{permID2}
	// GetVSOAPermissions also returns remaining perm IDs (used by computeVSOAFeegrantExpiration)
	delKeeper.GetVSOAPermissionsResult = []uint64{permID2}

	delKeeper.Reset()

	_, err = ms.RevokeParticipant(ctx, &types.MsgRevokeParticipant{
		Corporation: authority,
		Operator:    authority,
		Id:          permID1,
	})
	require.NoError(t, err)

	// RemovePermFromVSOA should be called
	require.Len(t, delKeeper.RemovePermFromVSOACalls, 1)

	// RevokeFeeAllowance should NOT be called (other perms remain)
	require.Len(t, delKeeper.RevokeFeeAllowanceCalls, 0)

	// GrantFeeAllowance should be called with recalculated expiration
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 1)
	require.Equal(t, authority, delKeeper.GrantFeeAllowanceCalls[0].Authority)
	require.Equal(t, vsOperator, delKeeper.GrantFeeAllowanceCalls[0].Grantee)
	// The expiration should be futureTime2 (the farthest remaining perm)
	require.NotNil(t, delKeeper.GrantFeeAllowanceCalls[0].Expiration)
	require.Equal(t, futureTime2.Unix(), delKeeper.GrantFeeAllowanceCalls[0].Expiration.Unix())
}

// TestVSOA_ComputeFeegrantExpirationReturnsNilForUnlimited verifies that when
// any remaining perm has no effective_until (unlimited), the feegrant expiration
// is nil (unlimited).
func TestVSOA_ComputeFeegrantExpirationReturnsNilForUnlimited(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(365 * 24 * time.Hour)

	// Perm #1: the one we will revoke (has expiry)
	permID1, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Perm #2: remaining perm with NO effective_until (unlimited)
	permID2, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              nil, // unlimited
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Configure mock
	delKeeper.RemovePermFromVSOARemainingPerms = []uint64{permID2}
	delKeeper.GetVSOAPermissionsResult = []uint64{permID2}

	delKeeper.Reset()

	_, err = ms.RevokeParticipant(ctx, &types.MsgRevokeParticipant{
		Corporation: authority,
		Operator:    authority,
		Id:          permID1,
	})
	require.NoError(t, err)

	// GrantFeeAllowance should be called with nil expiration (unlimited)
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 1)
	require.Nil(t, delKeeper.GrantFeeAllowanceCalls[0].Expiration)
}

// TestVSOA_ComputeFeegrantExpirationReturnsMaxExpiry verifies that when
// multiple remaining perms have different effective_until values, the feegrant
// expiration is set to the farthest (maximum) value.
func TestVSOA_ComputeFeegrantExpirationReturnsMaxExpiry(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	_ = trkKeeper
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	authority := sdk.AccAddress([]byte("test_authority______")).String()
	vsOperator := sdk.AccAddress([]byte("test_vs_operator____")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime1 := now.Add(100 * 24 * time.Hour) // ~100 days
	futureTime2 := now.Add(200 * 24 * time.Hour) // ~200 days
	futureTime3 := now.Add(500 * 24 * time.Hour) // ~500 days (max)
	futureTimeRevoked := now.Add(365 * 24 * time.Hour)

	// Perm #1: the one we will revoke
	permID1, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTimeRevoked,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Perm #2: remaining, expires in 100 days
	permID2, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime1,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Perm #3: remaining, expires in 200 days
	permID3, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime2,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Perm #4: remaining, expires in 500 days (the max)
	permID4, err := k.CreatePermission(sdkCtx, types.Participant{
		SchemaId:                    1,
		Role:                        types.ParticipantRole_ISSUER,
		CorporationId:               trkKeeper.RegisterCorp(authority),
		Created:                     &now,
		Adjusted:                    &now,
		Modified:                    &now,
		OpState:                     types.OnboardingState_VALIDATED,
		EffectiveFrom:               &pastTime,
		EffectiveUntil:              &futureTime3,
		VsOperator:                  vsOperator,
		VsOperatorAuthzEnabled:      true,
		VsOperatorAuthzWithFeegrant: true,
	})
	require.NoError(t, err)

	// Configure mock
	delKeeper.RemovePermFromVSOARemainingPerms = []uint64{permID2, permID3, permID4}
	delKeeper.GetVSOAPermissionsResult = []uint64{permID2, permID3, permID4}

	delKeeper.Reset()

	_, err = ms.RevokeParticipant(ctx, &types.MsgRevokeParticipant{
		Corporation: authority,
		Operator:    authority,
		Id:          permID1,
	})
	require.NoError(t, err)

	// GrantFeeAllowance should be called with the maximum expiry (500 days)
	require.Len(t, delKeeper.GrantFeeAllowanceCalls, 1)
	require.NotNil(t, delKeeper.GrantFeeAllowanceCalls[0].Expiration)
	require.Equal(t, futureTime3.Unix(), delKeeper.GrantFeeAllowanceCalls[0].Expiration.Unix())
}
