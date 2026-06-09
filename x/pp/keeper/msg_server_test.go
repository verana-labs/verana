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

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockCredentialSchemaKeeper, *keepertest.MockParticipantEcosystemKeeper, context.Context) {
	k, csKeeper, trkKeeper, _, ctx, _ := keepertest.ParticipantKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx
}

func setupMsgServerWithDelegation(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockCredentialSchemaKeeper, *keepertest.MockParticipantEcosystemKeeper, context.Context, *keepertest.MockDelegationKeeper) {
	k, csKeeper, trkKeeper, _, ctx, delKeeper := keepertest.ParticipantKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx, delKeeper
}

func TestMsgServer(t *testing.T) {
	k, ms, _, _, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

// Test for StartParticipantOP
func TestStartParticipantVP(t *testing.T) {
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

	// Create mock credential schema with specific participant management modes
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

	// Create validator participant (ISSUER_GRANTOR)
	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE
	// This should be VALIDATED as it's a prerequisite
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED, // validator must be validated
		EffectiveFrom: &pastTime,                       // Required for ACTIVE state
	}

	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create another validator participant (VERIFIER_GRANTOR with different country)
	verifierGrantorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_VERIFIER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	verifierGrantorParticipantID, err := k.CreateParticipant(sdkCtx, verifierGrantorParticipant)
	require.NoError(t, err)

	// Create a validator participant without country (for testing optional country)
	validatorParticipantNoCountry := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	validatorParticipantNoCountryID, err := k.CreateParticipant(sdkCtx, validatorParticipantNoCountry)
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
				ValidatorParticipantId: validatorParticipantID,
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
				ValidatorParticipantId: validatorParticipantID,
				Did:                    "did:example:start-fees-optional",
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
				ValidatorParticipantId: validatorParticipantID,
				Did:                    "did:example:start-fees-partial",
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
				ValidatorParticipantId: validatorParticipantID,
				Did:                    "did:example:start-fees-zero",
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
				ValidatorParticipantId: validatorParticipantNoCountryID,
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
			err:       "validator participant not found",
			checkFees: false,
		},
		{
			name: "Invalid Participant Type Combination - ISSUER with wrong validator",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: verifierGrantorParticipantID, // Wrong validator type
				Did:                    validDid,
			},
			err:       "issuer participant requires ISSUER_GRANTOR validator",
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

				// Verify created participant
				participant, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Role, participant.Role)
				require.NotZero(t, participant.CorporationId)
				require.Equal(t, tc.msg.ValidatorParticipantId, participant.ValidatorParticipantId)
				require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
				require.NotNil(t, participant.Created)
				require.NotNil(t, participant.Modified)
				require.NotNil(t, participant.OpLastStateChange)

				// Verify requested fees if provided
				if tc.checkFees {
					require.Equal(t, tc.expectedValidationFees, participant.ValidationFees, "Validation fees should match requested value")
					require.Equal(t, tc.expectedIssuanceFees, participant.IssuanceFees, "Issuance fees should match requested value")
					require.Equal(t, tc.expectedVerificationFees, participant.VerificationFees, "Verification fees should match requested value")
				} else {
					// If fees were not provided, they should be 0
					require.Equal(t, uint64(0), participant.ValidationFees, "Validation fees should be 0 when not provided")
					require.Equal(t, uint64(0), participant.IssuanceFees, "Issuance fees should be 0 when not provided")
					require.Equal(t, uint64(0), participant.VerificationFees, "Verification fees should be 0 when not provided")
				}
			}
		})
	}
}

func TestRenewParticipantVP(t *testing.T) {
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

	// Create validator participant
	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          3, // ISSUER_GRANTOR
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}

	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)

	require.NoError(t, err)

	// Create applicant participant
	applicantParticipant := types.Participant{
		SchemaId:               1,
		Role:                   1, // ISSUER
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	applicantParticipantID, err := k.CreateParticipant(sdk.UnwrapSDKContext(ctx), applicantParticipant)
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
			err: "participant not found",
		},
		{
			name: "Wrong Authority",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("wrong_creator")).String(),
				Operator:    sdk.AccAddress([]byte("wrong_creator")).String(),
				Id:          applicantParticipantID,
			},
			err: "authority is not the participant authority",
		},
		{
			name: "Successful Renewal",
			msg: &types.MsgRenewParticipantOP{
				Corporation: creator,
				Operator:    creator,
				Id:          applicantParticipantID,
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

				// Verify updated participant
				participant, err := k.GetParticipantByID(sdk.UnwrapSDKContext(ctx), tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
				require.NotNil(t, participant.OpLastStateChange)
			}
		})
	}
}

func TestRenewParticipantVP_AuthzCheck(t *testing.T) {
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

	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          3,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	applicantParticipant := types.Participant{
		SchemaId:               1,
		Role:                   1,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
	require.NoError(t, err)

	t.Run("AUTHZ-CHECK failure blocks renewal", func(t *testing.T) {
		mockDelegation.ErrToReturn = fmt.Errorf("operator not authorized")
		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantParticipantID,
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
			Id:          applicantParticipantID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
	})
}

func TestRenewParticipantVP_VpStatePrecondition(t *testing.T) {
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

	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          3,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	t.Run("Renewing PENDING participant is blocked (prevents fee accounting loss)", func(t *testing.T) {
		pendingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          1000, // funds already in escrow
			OpCurrentDeposit:       500,
		}
		pendingParticipantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          pendingParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "op_state must be VALIDATED to renew")
		require.Nil(t, resp)

		// Verify the participant was NOT modified (fees still intact)
		participant, err := k.GetParticipantByID(sdkCtx, pendingParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
		require.Equal(t, uint64(1000), participant.OpCurrentFees)
		require.Equal(t, uint64(500), participant.OpCurrentDeposit)
	})

	t.Run("Renewing UNSPECIFIED op_state participant is blocked", func(t *testing.T) {
		unspecParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_ONBOARDING_STATE_UNSPECIFIED,
		}
		unspecParticipantID, err := k.CreateParticipant(sdkCtx, unspecParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          unspecParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "op_state must be VALIDATED to renew")
		require.Nil(t, resp)
	})

	t.Run("Renewing VALIDATED participant succeeds", func(t *testing.T) {
		validatedParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		validatedParticipantID, err := k.CreateParticipant(sdkCtx, validatedParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          validatedParticipantID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, validatedParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
		require.NotNil(t, participant.OpLastStateChange)
		require.Equal(t, now, *participant.OpLastStateChange)
		require.Equal(t, now, *participant.Modified)
	})
}

func TestRenewParticipantVP_ValidatorParticipantChecks(t *testing.T) {
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

	t.Run("Renewal blocked when validator participant is revoked", func(t *testing.T) {
		revokedTime := now.Add(-30 * time.Minute)
		revokedValidatorParticipant := types.Participant{
			SchemaId:      1,
			Role:          3,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
			Revoked:       &revokedTime,
		}
		revokedValParticipantID, err := k.CreateParticipant(sdkCtx, revokedValidatorParticipant)
		require.NoError(t, err)

		applicantParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: revokedValParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator participant is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator participant is expired", func(t *testing.T) {
		expiredTime := now.Add(-10 * time.Minute)
		expiredValidatorParticipant := types.Participant{
			SchemaId:       1,
			Role:           3,
			CorporationId:  trkKeeper.RegisterCorp(creator),
			Created:        &now,
			Modified:       &now,
			OpState:        types.OnboardingState_VALIDATED,
			EffectiveFrom:  &pastTime,
			EffectiveUntil: &expiredTime,
		}
		expiredValParticipantID, err := k.CreateParticipant(sdkCtx, expiredValidatorParticipant)
		require.NoError(t, err)

		applicantParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: expiredValParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator participant is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator participant is INACTIVE (no effective_from)", func(t *testing.T) {
		inactiveValidatorParticipant := types.Participant{
			SchemaId:      1,
			Role:          3,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			// EffectiveFrom is nil => INACTIVE
		}
		inactiveValParticipantID, err := k.CreateParticipant(sdkCtx, inactiveValidatorParticipant)
		require.NoError(t, err)

		applicantParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: inactiveValParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator participant is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator participant does not exist", func(t *testing.T) {
		applicantParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: 99999, // non-existent
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator participant not found")
		require.Nil(t, resp)
	})
}

func TestRenewParticipantVP_FeeAndDepositAccumulation(t *testing.T) {
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

	validatorParticipant := types.Participant{
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
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	t.Run("Deposit accumulates on renewal", func(t *testing.T) {
		initialDeposit := uint64(100)
		applicantParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			Deposit:                initialDeposit,
		}
		applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    creator,
			Id:          applicantParticipantID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
		// Deposit should accumulate: initialDeposit + new deposit
		require.True(t, participant.Deposit >= initialDeposit, "deposit should accumulate, got %d", participant.Deposit)
		require.True(t, participant.OpCurrentFees > 0 || participant.OpCurrentDeposit > 0 || validatorParticipant.ValidationFees == 0,
			"current fees or deposit should be set based on validator fees")
	})

	t.Run("Different operator than authority is allowed", func(t *testing.T) {
		operator := sdk.AccAddress([]byte("different_oper")).String()
		applicantParticipant := types.Participant{
			SchemaId:               1,
			Role:                   1,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
		require.NoError(t, err)

		resp, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{
			Corporation: creator,
			Operator:    operator,
			Id:          applicantParticipantID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
	})
}

func TestRenewParticipantVP_ValidateBasic(t *testing.T) {
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
			name: "Zero participant ID",
			msg: &types.MsgRenewParticipantOP{
				Corporation: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:    sdk.AccAddress([]byte("test_operator")).String(),
				Id:          0,
			},
			err: "participant ID cannot be 0",
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

func TestSetParticipantVPToValidated(t *testing.T) {
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

	// Create validator participant
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}

	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// 1. Test with new participant (not renewal case)
	t.Run("Valid new participant validation", func(t *testing.T) {
		// Create a new participant in PENDING state
		newParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		newParticipantID, err := k.CreateParticipant(sdkCtx, newParticipant)
		require.NoError(t, err)

		// Set participant to validated
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      newParticipantID,
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

		// Verify participant was updated correctly
		updatedParticipant, err := k.GetParticipantByID(sdkCtx, newParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_VALIDATED, updatedParticipant.OpState)
		require.Equal(t, msg.ValidationFees, updatedParticipant.ValidationFees)
		require.Equal(t, msg.IssuanceFees, updatedParticipant.IssuanceFees)
		require.Equal(t, msg.VerificationFees, updatedParticipant.VerificationFees)
		require.Equal(t, msg.IssuanceFeeDiscount, updatedParticipant.IssuanceFeeDiscount)
		require.Equal(t, msg.VerificationFeeDiscount, updatedParticipant.VerificationFeeDiscount)
		require.NotNil(t, updatedParticipant.EffectiveFrom)
		require.Equal(t, now.Unix(), updatedParticipant.EffectiveFrom.Unix()) // First time: set to now
		require.NotNil(t, updatedParticipant.EffectiveUntil)
		require.Equal(t, futureTime.Unix(), updatedParticipant.EffectiveUntil.Unix())
		require.Equal(t, msg.OpSummaryDigest, updatedParticipant.OpSummaryDigest)
		// Execution assertions
		require.NotNil(t, updatedParticipant.Modified)
		require.Equal(t, now.Unix(), updatedParticipant.Modified.Unix())
		require.NotNil(t, updatedParticipant.OpLastStateChange)
		require.Equal(t, now.Unix(), updatedParticipant.OpLastStateChange.Unix())
		require.Equal(t, uint64(0), updatedParticipant.OpCurrentFees)    // Reset to 0
		require.Equal(t, uint64(0), updatedParticipant.OpCurrentDeposit) // Reset to 0
	})

	// 2. Test renewal case - participant already has EffectiveFrom
	t.Run("Renewal participant validation", func(t *testing.T) {
		renewalAddr := sdk.AccAddress([]byte("renewal_creator")).String()
		// Create a participant that already has EffectiveFrom set (renewal)
		effectiveFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
		currentEffectiveUntil := now.Add(-1 * time.Hour)
		renewalParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(renewalAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffectiveUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		renewalParticipantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		// Set participant to validated with same fees
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      renewalParticipantID,
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

		// Verify participant was updated correctly
		updatedParticipant, err := k.GetParticipantByID(sdkCtx, renewalParticipantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_VALIDATED, updatedParticipant.OpState)
		// Fees should remain unchanged (renewal doesn't overwrite)
		require.Equal(t, renewalParticipant.ValidationFees, updatedParticipant.ValidationFees)
		require.Equal(t, renewalParticipant.IssuanceFees, updatedParticipant.IssuanceFees)
		require.Equal(t, renewalParticipant.VerificationFees, updatedParticipant.VerificationFees)
		// EffectiveFrom should NOT change on renewal
		require.Equal(t, effectiveFrom.Unix(), updatedParticipant.EffectiveFrom.Unix())
		require.NotNil(t, updatedParticipant.EffectiveUntil)
		require.Equal(t, futureTime.Unix(), updatedParticipant.EffectiveUntil.Unix())
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
		require.Contains(t, err.Error(), "participant not found")
		require.Nil(t, resp)
	})

	// 4. Test validation error - Not in PENDING state
	t.Run("Not in PENDING state", func(t *testing.T) {
		// Create a participant that's not in PENDING state
		notPendingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED, // Not PENDING
		}
		notPendingParticipantID, err := k.CreateParticipant(sdkCtx, notPendingParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation: validatorAddr,
			Operator:    validatorAddr,
			Id:          notPendingParticipantID,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "participant must be in PENDING state")
		require.Nil(t, resp)
	})

	// 5. Test validation error - Wrong validator
	t.Run("Wrong validator", func(t *testing.T) {
		// Create a new participant in PENDING state
		pendingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		pendingParticipantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation: otherAddr, // Not the validator
			Operator:    otherAddr,
			Id:          pendingParticipantID,
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority must be validator participant authority")
		require.Nil(t, resp)
	})

	// 6. Test validation error - HOLDER with digest SRI
	t.Run("HOLDER type with digest SRI", func(t *testing.T) {
		// Create a HOLDER participant in PENDING state
		holderParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_HOLDER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		holderParticipantID, err := k.CreateParticipant(sdkCtx, holderParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      holderParticipantID,
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
		// Create ISSUER_GRANTOR participant in PENDING state
		grantorParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		grantorParticipantID, err := k.CreateParticipant(sdkCtx, grantorParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      grantorParticipantID,
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

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, grantorParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), updatedParticipant.IssuanceFeeDiscount)
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
		validatorWithDiscountID, err := k.CreateParticipant(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		// Create ISSUER participant with this validator
		issuerParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorWithDiscountID,
			OpState:                types.OnboardingState_PENDING,
		}
		issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
		require.NoError(t, err)

		// Can set discount up to validator's discount (7000)
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerParticipantID,
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

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, issuerParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), updatedParticipant.IssuanceFeeDiscount)
	})

	// 9. Test discount validation - ISSUER in GRANTOR mode exceeding validator's discount
	t.Run("ISSUER in GRANTOR mode exceeding validator discount", func(t *testing.T) {
		// Create ISSUER participant with validator that has 50% discount
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
		validatorWithDiscountID, err := k.CreateParticipant(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		issuerParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorWithDiscountID,
			OpState:                types.OnboardingState_PENDING,
		}
		issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
		require.NoError(t, err)

		// Try to set discount exceeding validator's discount
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerParticipantID,
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
		grantorParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		grantorParticipantID, err := k.CreateParticipant(sdkCtx, grantorParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      grantorParticipantID,
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
		renewalParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(otherAddr), // Use different authority to avoid overlap with test 7
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
			IssuanceFeeDiscount:    3000, // 30% discount set initially
		}
		renewalParticipantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		// Try to change discount during renewal
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      renewalParticipantID,
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

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, renewalParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(3000), updatedParticipant.IssuanceFeeDiscount)
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
		ecosystemValidatorID, err := k.CreateParticipant(sdkCtx, ecosystemValidator)
		require.NoError(t, err)

		// Create ISSUER participant with ECOSYSTEM validator
		issuerParticipant := types.Participant{
			SchemaId:               2,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: ecosystemValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerParticipantID,
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

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, issuerParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(8000), updatedParticipant.IssuanceFeeDiscount)
	})

	// 13. Test effective_until <= now (first time) should fail
	t.Run("effective_until must be greater than now for first time", func(t *testing.T) {
		euAddr := sdk.AccAddress([]byte("eu_now_creator")).String()
		pendingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(euAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		participantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
		require.NoError(t, err)

		pastEffUntil := now.Add(-1 * time.Hour) // in the past
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
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
		vpValidatorID, err := k.CreateParticipant(sdkCtx, vpValidator)
		require.NoError(t, err)

		vpTestAddr := sdk.AccAddress([]byte("op_exp_creator")).String()
		pendingParticipant := types.Participant{
			SchemaId:               3,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(vpTestAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vpValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		participantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
		require.NoError(t, err)

		// vpExp will be now + 30 days. Set effective_until to now + 60 days (beyond vpExp)
		farFuture := now.Add(60 * 24 * time.Hour)
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      vpAddr,
			Operator:         vpAddr,
			Id:               participantID,
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
		// Find the validator participant ID for schema 3
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
		vpValidator2ID, err := k.CreateParticipant(sdkCtx, vpValidator2)
		require.NoError(t, err)

		pendingParticipant := types.Participant{
			SchemaId:               3,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(vpNilAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vpValidator2ID,
			OpState:                types.OnboardingState_PENDING,
		}
		participantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      vpAddr,
			Operator:         vpAddr,
			Id:               participantID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   nil, // nil should resolve to vpExp
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, participantID)
		require.NoError(t, err)
		// effective_until should equal vpExp (now + 30 days)
		expectedVpExp := now.AddDate(0, 0, 30)
		require.NotNil(t, updatedParticipant.OpExp)
		require.Equal(t, expectedVpExp.Unix(), updatedParticipant.OpExp.Unix())
		require.NotNil(t, updatedParticipant.EffectiveUntil)
		require.Equal(t, expectedVpExp.Unix(), updatedParticipant.EffectiveUntil.Unix())
	})

	// 16. Test renewal effective_until must be greater than current effective_until
	t.Run("Renewal effective_until must be greater than current", func(t *testing.T) {
		renewAddr := sdk.AccAddress([]byte("renew_eu_creator")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(30 * 24 * time.Hour) // 30 days in future
		renewalParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(renewAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		participantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		// Try with effective_until <= current effective_until
		smallerEffUntil := now.Add(10 * 24 * time.Hour)
		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
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
		renewalParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rvfAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		participantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
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
		renewalParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rifAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		participantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
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
		renewalParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rvAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			EffectiveFrom:          &effectiveFrom,
			EffectiveUntil:         &currentEffUntil,
			ValidationFees:         10,
			IssuanceFees:           5,
			VerificationFees:       3,
		}
		participantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
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

	// 20. Test overlap - existing participant never expires (nil effective_until)
	t.Run("Overlap with never-expiring participant", func(t *testing.T) {
		overlapAddr := sdk.AccAddress([]byte("overlap_never_addr")).String()
		// Create an existing validated participant with nil effective_until (never expires)
		existingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         nil, // Never expires
		}
		_, err := k.CreateParticipant(sdkCtx, existingParticipant)
		require.NoError(t, err)

		// Try to create a new validated participant with same (schema, type, validator, authority)
		newParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		newParticipantID, err := k.CreateParticipant(sdkCtx, newParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               newParticipantID,
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

	// 21. Test overlap - existing participant's effective_until after new participant's effective_from
	t.Run("Overlap with active participant time range", func(t *testing.T) {
		overlapAddr2 := sdk.AccAddress([]byte("overlap_range_addr")).String()
		// Create an existing validated participant that's still active (effective_until in the future)
		existingEffUntil := now.Add(30 * 24 * time.Hour) // 30 days from now
		existingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr2),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         &existingEffUntil,
		}
		_, err := k.CreateParticipant(sdkCtx, existingParticipant)
		require.NoError(t, err)

		// Try to validate a new participant — effective_from will be set to now, which is before existing's effective_until
		newParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(overlapAddr2),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		newParticipantID, err := k.CreateParticipant(sdkCtx, newParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               newParticipantID,
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

	// 22. Test validator participant not active (revoked)
	t.Run("Validator participant is revoked", func(t *testing.T) {
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
		revokedValidatorID, err := k.CreateParticipant(sdkCtx, revokedValidator)
		require.NoError(t, err)

		rvAddr := sdk.AccAddress([]byte("revoked_val_addr")).String()
		pendingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(rvAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: revokedValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		participantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator participant is not valid")
		require.Nil(t, resp)
	})

	// 23. Test with OpCurrentFees and OpCurrentDeposit > 0 (execution: fee transfer + trust deposit)
	t.Run("Execution with fees and trust deposit", func(t *testing.T) {
		feeAddr := sdk.AccAddress([]byte("fee_exec_creator")).String()
		newParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(feeAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          100, // Has fees to transfer
			OpCurrentDeposit:       50,  // Has deposit to transfer
		}
		participantID, err := k.CreateParticipant(sdkCtx, newParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               participantID,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
			EffectiveUntil:   &futureTime,
			OpSummaryDigest:  "sha384-validDigest",
		}

		resp, err := ms.SetParticipantOPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, participantID)
		require.NoError(t, err)
		require.Equal(t, uint64(0), updatedParticipant.OpCurrentFees)       // Reset to 0
		require.Equal(t, uint64(0), updatedParticipant.OpCurrentDeposit)    // Reset to 0
		require.Equal(t, uint64(50), updatedParticipant.OpValidatorDeposit) // Accumulated
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
		vgValidatorID, err := k.CreateParticipant(sdkCtx, vgValidator)
		require.NoError(t, err)

		// Create VERIFIER_GRANTOR participant (can set its own verification_fee_discount)
		vgParticipant := types.Participant{
			SchemaId:               4,
			Role:                   types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId:          trkKeeper.RegisterCorp(sdk.AccAddress([]byte("vg_participant_creator")).String()),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vgValidatorID,
			OpState:                types.OnboardingState_PENDING,
		}
		vgParticipantID, err := k.CreateParticipant(sdkCtx, vgParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             vgAddr,
			Operator:                vgAddr,
			Id:                      vgParticipantID,
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

		updatedParticipant, err := k.GetParticipantByID(sdkCtx, vgParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(6000), updatedParticipant.VerificationFeeDiscount)
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
		vgValidator2ID, err := k.CreateParticipant(sdkCtx, vgValidator2)
		require.NoError(t, err)

		verParticipant := types.Participant{
			SchemaId:               4,
			Role:                   types.ParticipantRole_VERIFIER,
			CorporationId:          trkKeeper.RegisterCorp(sdk.AccAddress([]byte("ver_exceed_addr")).String()),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: vgValidator2ID,
			OpState:                types.OnboardingState_PENDING,
		}
		verParticipantID, err := k.CreateParticipant(sdkCtx, verParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             vgAddr2,
			Operator:                vgAddr2,
			Id:                      verParticipantID,
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

	// 26. Test overlap check skips revoked participants
	t.Run("Overlap check skips revoked participants", func(t *testing.T) {
		skipAddr := sdk.AccAddress([]byte("overlap_skip_addr")).String()
		revokedTime := now.Add(-1 * time.Hour)
		// Create a revoked participant (should be skipped in overlap check)
		revokedParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(skipAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
			EffectiveUntil:         nil, // Never expires, but it's revoked
			Revoked:                &revokedTime,
		}
		_, err := k.CreateParticipant(sdkCtx, revokedParticipant)
		require.NoError(t, err)

		// This new participant should NOT conflict with the revoked one
		newParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(skipAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		newParticipantID, err := k.CreateParticipant(sdkCtx, newParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:      validatorAddr,
			Operator:         validatorAddr,
			Id:               newParticipantID,
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
		grantorParticipant := types.Participant{
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
				id, _ := k.CreateParticipant(sdkCtx, v)
				return id
			}(),
			OpState: types.OnboardingState_PENDING,
		}
		grantorParticipantID, err := k.CreateParticipant(sdkCtx, grantorParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             sdk.AccAddress([]byte("vfd_max_validato")).String(),
			Operator:                sdk.AccAddress([]byte("vfd_max_validato")).String(),
			Id:                      grantorParticipantID,
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
		renewalParticipant := types.Participant{
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
				id, _ := k.CreateParticipant(sdkCtx, v)
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
		participantID, err := k.CreateParticipant(sdkCtx, renewalParticipant)
		require.NoError(t, err)

		msg := &types.MsgSetParticipantOPToValidated{
			Corporation:             sdk.AccAddress([]byte("rvd_validator_ad")).String(),
			Operator:                sdk.AccAddress([]byte("rvd_validator_ad")).String(),
			Id:                      participantID,
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
func TestSetParticipantVPToValidated_AuthzCheckFailure(t *testing.T) {
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

	// Create validator participant
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create participant to validate
	pendingParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creatorAddr),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_PENDING,
	}
	participantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
	require.NoError(t, err)

	// Set delegation keeper to return error
	delKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

	msg := &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         operatorAddr,
		Id:               participantID,
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

	// Need to use validatorAddr as authority (not creatorAddr) since validator participant authority check
	resp, err = ms.SetParticipantOPToValidated(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

// TestMsgServerCreateRootParticipant is superseded by TestCreateRootParticipant which has
// comprehensive coverage of all spec v4 checks including overlap and AUTHZ.
// Keeping as a simple smoke test with updated field names.
func TestMsgServerCreateRootParticipant(t *testing.T) {
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

	participant, err := k.GetParticipantByID(sdkCtx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, uint64(1), participant.SchemaId)
	require.Equal(t, validDid, participant.Did)
	require.NotZero(t, participant.CorporationId)
	// [MOD-PP-MSG-7-3] spec v4 draft 13: participant.type is hardcoded to ECOSYSTEM.
	require.Equal(t, types.ParticipantRole_ECOSYSTEM, participant.Role)
	require.Equal(t, uint64(100), participant.ValidationFees)
	require.Equal(t, uint64(50), participant.IssuanceFees)
	require.Equal(t, uint64(25), participant.VerificationFees)
	require.Equal(t, uint64(0), participant.Deposit)
	require.NotNil(t, participant.Created)
	require.NotNil(t, participant.Modified)
}

func TestCancelParticipantVPLastRequest(t *testing.T) {
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

	// Create validator participant
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// [MOD-PP-MSG-6-3] Spec v4 draft 13: when op_exp is null (never validated),
	// set op_state to TERMINATED. The participant row is retained.
	t.Run("Valid cancellation - never validated before", func(t *testing.T) {
		neverAddr := sdk.AccAddress([]byte("never_val_cancel")).String()
		neverValidatedParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(neverAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		participantID, err := k.CreateParticipant(sdkCtx, neverValidatedParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: neverAddr,
			Operator:    neverAddr,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Participant is retained and transitioned to TERMINATED.
		got, err := k.GetParticipantByID(sdkCtx, participantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_TERMINATED, got.OpState)
	})

	// 2. Valid cancellation - previously validated (renewal: EffectiveFrom set → VALIDATED)
	t.Run("Valid cancellation - previously validated", func(t *testing.T) {
		prevAddr := sdk.AccAddress([]byte("prev_val_cancel")).String()
		pastTime := now.Add(-1 * time.Hour)
		futureTime := now.Add(24 * time.Hour)
		previouslyValidatedParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(prevAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			OpExp:                  &futureTime, // Has a previous validation
			EffectiveFrom:          &pastTime,   // Renewal: was previously activated
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		participantID, err := k.CreateParticipant(sdkCtx, previouslyValidatedParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: prevAddr,
			Operator:    prevAddr,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, participantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_VALIDATED, participant.OpState)
		require.Equal(t, uint64(0), participant.OpCurrentFees)
		require.Equal(t, uint64(0), participant.OpCurrentDeposit)
	})

	// 3. Invalid - participant not found
	t.Run("Invalid - participant not found", func(t *testing.T) {
		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: creator,
			Operator:    creator,
			Id:          9999,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "participant not found")
		require.Nil(t, resp)
	})

	// 4. Invalid - wrong authority
	t.Run("Invalid - wrong authority", func(t *testing.T) {
		wrongAuthParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
		}
		participantID, err := k.CreateParticipant(sdkCtx, wrongAuthParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: otherAddr, // Not the participant authority
			Operator:    otherAddr,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not the participant authority")
		require.Nil(t, resp)
	})

	// 5. Invalid - not in PENDING state
	t.Run("Invalid - not in PENDING state", func(t *testing.T) {
		notPendingParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
		}
		participantID, err := k.CreateParticipant(sdkCtx, notPendingParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: creator,
			Operator:    creator,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "participant must be in PENDING state")
		require.Nil(t, resp)
	})

	// 6. Invalid - slashed and not repaid
	t.Run("Invalid - slashed and not repaid", func(t *testing.T) {
		slashedTime := now.Add(-1 * time.Hour)
		slashedParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(creator),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			Slashed:                &slashedTime, // Slashed
			// Repaid is nil (not repaid)
		}
		participantID, err := k.CreateParticipant(sdkCtx, slashedParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: creator,
			Operator:    creator,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "slashed and not repaid")
		require.Nil(t, resp)
	})

	// 7. Valid - slashed but repaid (allowed), first-time VP → participant deleted
	t.Run("Valid - slashed and repaid is allowed", func(t *testing.T) {
		repaidAddr := sdk.AccAddress([]byte("repaid_cancel_ad")).String()
		slashedTime := now.Add(-2 * time.Hour)
		repaidTime := now.Add(-1 * time.Hour)
		repaidParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(repaidAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			Slashed:                &slashedTime,
			Repaid:                 &repaidTime, // Repaid
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		participantID, err := k.CreateParticipant(sdkCtx, repaidParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: repaidAddr,
			Operator:    repaidAddr,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// [MOD-PP-MSG-6-3] Never-validated participant transitions to TERMINATED; row retained.
		got, err := k.GetParticipantByID(sdkCtx, participantID)
		require.NoError(t, err)
		require.Equal(t, types.OnboardingState_TERMINATED, got.OpState)
	})

	// 8. Valid cancellation with zero fees (no transfer needed)
	t.Run("Valid cancellation with zero fees", func(t *testing.T) {
		zeroFeesAddr := sdk.AccAddress([]byte("zero_fees_cancel")).String()
		zeroFeesParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(zeroFeesAddr),
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_PENDING,
			OpCurrentFees:          0,
			OpCurrentDeposit:       0,
		}
		participantID, err := k.CreateParticipant(sdkCtx, zeroFeesParticipant)
		require.NoError(t, err)

		msg := &types.MsgCancelParticipantOPLastRequest{
			Corporation: zeroFeesAddr,
			Operator:    zeroFeesAddr,
			Id:          participantID,
		}

		resp, err := ms.CancelParticipantOPLastRequest(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestCancelParticipantVPLastRequest_AuthzCheckFailure tests AUTHZ-CHECK for CancelParticipantOPLastRequest
func TestCancelParticipantVPLastRequest_AuthzCheckFailure(t *testing.T) {
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

	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	pendingParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(creatorAddr),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_PENDING,
	}
	participantID, err := k.CreateParticipant(sdkCtx, pendingParticipant)
	require.NoError(t, err)

	// Set delegation keeper to return error
	delKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

	msg := &types.MsgCancelParticipantOPLastRequest{
		Corporation: creatorAddr,
		Operator:    operatorAddr,
		Id:          participantID,
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

// TestAdjustParticipant tests the SetParticipantEffectiveUntil message server function
func TestAdjustParticipant(t *testing.T) {
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

	// Create distinct mock credential schemas to avoid overlap between test participants.
	// Each participant uses a unique schema_id so the overlap check doesn't fire across test cases.
	for i := uint64(1); i <= 10; i++ {
		csKeeper.CreateMockCredentialSchema(i,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	}

	now := sdkCtx.BlockTime()
	currentEffectiveUntil := now.Add(30 * 24 * time.Hour) // 30 days in the future
	futureVpExp := now.Add(365 * 24 * time.Hour)          // 1 year in the future
	pastTime := now.Add(-1 * time.Hour)                   // Set effective_from to past to make it ACTIVE

	// Create validator participant (ISSUER_GRANTOR) — schema 1
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create a VP managed participant to adjust — schema 2
	applicantParticipant := types.Participant{
		SchemaId:               2,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         &currentEffectiveUntil,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
	require.NoError(t, err)

	// Create an ECOSYSTEM participant — schema 3
	ecosystemParticipant := types.Participant{
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
	ecosystemParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	// Create a participant for the "wrong authority" test — schema 4
	wrongAuthTestParticipant := types.Participant{
		SchemaId:               4,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         &currentEffectiveUntil,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	wrongAuthTestParticipantID, err := k.CreateParticipant(sdkCtx, wrongAuthTestParticipant)
	require.NoError(t, err)

	// Create a participant with NULL effective_until — schema 5
	nullEffectiveUntilParticipant := types.Participant{
		SchemaId:               5,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         nil,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	nullEffectiveUntilParticipantID, err := k.CreateParticipant(sdkCtx, nullEffectiveUntilParticipant)
	require.NoError(t, err)

	// Create an ecosystem participant with NULL effective_until — schema 6
	nullEffectiveUntilEcosystemParticipant := types.Participant{
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
	nullEffectiveUntilEcosystemParticipantID, err := k.CreateParticipant(sdkCtx, nullEffectiveUntilEcosystemParticipant)
	require.NoError(t, err)

	// Create participant for past effective_until test — schema 7
	nullEffUntilPastTestParticipant := types.Participant{
		SchemaId:               7,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		EffectiveUntil:         nil,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	nullEffUntilPastTestParticipantID, err := k.CreateParticipant(sdkCtx, nullEffUntilPastTestParticipant)
	require.NoError(t, err)

	// Create participant for reduce effective_until test — schema 8
	reduceParticipant := types.Participant{
		SchemaId:               8,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Modified:               &now,
		EffectiveUntil:         &currentEffectiveUntil, // 30 days
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		OpExp:                  &futureVpExp,
		EffectiveFrom:          &pastTime,
	}
	reduceParticipantID, err := k.CreateParticipant(sdkCtx, reduceParticipant)
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
				Id:             applicantParticipantID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Valid adjustment by ecosystem authority",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    ecosystemAddr,
				Operator:       operatorAddr,
				Id:             ecosystemParticipantID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Invalid - participant not found",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             9999,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "participant not found",
		},
		{
			name: "Invalid - effective_until in the past",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             applicantParticipantID,
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
				Id:             applicantParticipantID,
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
				Id:             applicantParticipantID,
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
				Id:             wrongAuthTestParticipantID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "authority is not the validator participant authority",
		},
		{
			name: "Valid - adjust participant with NULL effective_until (VP managed)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             nullEffectiveUntilParticipantID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Valid - adjust participant with NULL effective_until (ecosystem)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    ecosystemAddr,
				Operator:       operatorAddr,
				Id:             nullEffectiveUntilEcosystemParticipantID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Invalid - effective_until in the past (NULL current effective_until)",
			msg: &types.MsgSetParticipantEffectiveUntil{
				Corporation:    validatorAddr,
				Operator:       operatorAddr,
				Id:             nullEffUntilPastTestParticipantID,
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
				Id:             reduceParticipantID,
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

				// Verify participant was adjusted
				participant, err := k.GetParticipantByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.EffectiveUntil.Unix(), participant.EffectiveUntil.Unix())
				require.NotNil(t, participant.Adjusted)
				require.NotNil(t, participant.Modified)
			}
		})
	}
}

// TestRevokeParticipant tests the RevokeParticipant message server function
func TestRevokeParticipant(t *testing.T) {
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

	// Create validator participant — schema 1
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create a participant to revoke — schema 2
	applicantParticipant := types.Participant{
		SchemaId:               2,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
	require.NoError(t, err)

	// Create another participant for the wrong-authority test — schema 2
	wrongAuthParticipant := types.Participant{
		SchemaId:               2,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	wrongAuthParticipantID, err := k.CreateParticipant(sdkCtx, wrongAuthParticipant)
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
				Id:          applicantParticipantID,
			},
			expectErr: false,
		},
		{
			name: "Invalid - participant not found",
			msg: &types.MsgRevokeParticipant{
				Corporation: validatorAddr,
				Operator:    operatorAddr,
				Id:          9999,
			},
			expectErr:  true,
			errMessage: "participant not found",
		},
		{
			name: "Invalid - wrong authority (not validator, not self, not TR controller)",
			msg: &types.MsgRevokeParticipant{
				Corporation: wrongAddr,
				Operator:    operatorAddr,
				Id:          wrongAuthParticipantID,
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

				// Verify participant was revoked
				participant, err := k.GetParticipantByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, participant.Revoked)
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

	// Create trust registry / ecosystem participant
	trustParticipant := types.Participant{
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
	trustParticipantID, err := k.CreateParticipant(sdkCtx, trustParticipant)
	require.NoError(t, err)

	issuerParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
	require.NoError(t, err)

	issuerParticipantNoAuthz := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerParticipantNoAuthzID, err := k.CreateParticipant(sdkCtx, issuerParticipantNoAuthz)
	require.NoError(t, err)

	// Create issuer participant with different vs_operator
	issuerParticipantDiffOp := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             otherOperator,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerParticipantDiffOpID, err := k.CreateParticipant(sdkCtx, issuerParticipantDiffOp)
	require.NoError(t, err)

	// Create issuer participant with different authority
	issuerParticipantDiffAuth := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(otherAuthority),
		VsOperator:             operator,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerParticipantDiffAuthID, err := k.CreateParticipant(sdkCtx, issuerParticipantDiffAuth)
	require.NoError(t, err)

	// Create verifier participant
	verifierParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_VERIFIER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	verifierParticipantID, err := k.CreateParticipant(sdkCtx, verifierParticipant)
	require.NoError(t, err)

	// Create agent participant
	agentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentParticipantID, err := k.CreateParticipant(sdkCtx, agentParticipant)
	require.NoError(t, err)

	// Create wallet agent participant
	walletAgentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	walletAgentParticipantID, err := k.CreateParticipant(sdkCtx, walletAgentParticipant)
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
				IssuerParticipantId:      issuerParticipantID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
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
				VerifierParticipantId:    verifierParticipantID,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr: false,
		},
		{
			name: "AUTHZ check failure",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerParticipantID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			setupErr:   fmt.Errorf("operator not authorized"),
			expectErr:  true,
			errMessage: "VS operator authorization check failed",
		},
		// Note: Invalid UUID is caught by ValidateBasic at SDK level, not in the handler
		{
			name: "Both participants missing",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      0,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr:  true,
			errMessage: "at least one of issuer_participant_id or verifier_participant_id must be provided",
		},
		{
			name: "Issuer participant not found",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      9999,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr:  true,
			errMessage: "issuer participant not found",
		},
		{
			name: "Issuer wrong type",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      trustParticipantID, // ECOSYSTEM type, not ISSUER
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr:  true,
			errMessage: "issuer participant must be ISSUER type",
		},
		{
			name: "Issuer vs_operator mismatch",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator, // does not match issuerParticipantDiffOp.VsOperator
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerParticipantDiffOpID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr:  true,
			errMessage: "issuer participant vs_operator does not match operator",
		},
		{
			name: "Issuer authority mismatch",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority, // does not match issuerParticipantDiffAuth.Authority
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerParticipantDiffAuthID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr:  true,
			errMessage: "issuer participant authority does not match authority",
		},
		{
			name: "VS operator authz check fails on participant",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerParticipantNoAuthzID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			setupErr:   fmt.Errorf("no VSOA record for participant"),
			expectErr:  true,
			errMessage: "VS operator authorization check failed",
		},
		{
			name: "Agent participant not found",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerParticipantID,
				VerifierParticipantId:    0,
				AgentParticipantId:       9999,
				WalletAgentParticipantId: walletAgentParticipantID,
			},
			expectErr:  true,
			errMessage: "agent participant not found",
		},
		{
			name: "Wallet agent participant not found",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              authority,
				Operator:                 operator,
				Id:                       uuid.New().String(),
				IssuerParticipantId:      issuerParticipantID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: 9999,
			},
			expectErr:  true,
			errMessage: "wallet agent participant not found",
		},
		{
			name: "Session update - authority mismatch",
			msg: &types.MsgCreateOrUpdateParticipantSession{
				Corporation:              otherAuthority, // different from session creator
				Operator:                 operator,
				Id:                       sessionUUID, // same ID as first test case (already created)
				IssuerParticipantId:      issuerParticipantDiffAuthID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
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
				IssuerParticipantId:      issuerParticipantID,
				VerifierParticipantId:    0,
				AgentParticipantId:       agentParticipantID,
				WalletAgentParticipantId: walletAgentParticipantID,
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

	// Create validator participant (ISSUER_GRANTOR) with issuance fees
	validatorParticipant := types.Participant{
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
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create ISSUER participant with discount set (per Issue #94: use discount instead of exemption)
	issuerParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		VsOperator:             operator,
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		IssuanceFeeDiscount:    5000, // 50% discount
		EffectiveFrom:          &pastTime,
	}
	issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
	require.NoError(t, err)

	// Create agent participant
	agentParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: issuerParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	agentParticipantID, err := k.CreateParticipant(sdkCtx, agentParticipant)
	require.NoError(t, err)

	walletAgentParticipantID := agentParticipantID // Use same for simplicity

	t.Run("Discount applied to beneficiary fees", func(t *testing.T) {
		// When creating a session with issuerParticipantID:
		// 1. Sum fees from found_participant_set (validatorParticipant with IssuanceFees=100)
		// 2. Apply exemption from issuerParticipant: beneficiary_fees = 100 * (1 - 0.5) = 50
		// Expected: beneficiary_fees = 50

		msg := &types.MsgCreateOrUpdateParticipantSession{
			Corporation:              authority,
			Operator:                 operator,
			Id:                       uuid.New().String(),
			IssuerParticipantId:      issuerParticipantID,
			VerifierParticipantId:    0,
			AgentParticipantId:       agentParticipantID,
			WalletAgentParticipantId: walletAgentParticipantID,
		}

		resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, msg.Id, resp.Id)
	})

	t.Run("Discount applied in execution", func(t *testing.T) {
		// Create another issuer participant with different discount
		issuerParticipant2 := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			VsOperator:             operator,
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			IssuanceFeeDiscount:    3000, // 30% discount
			EffectiveFrom:          &pastTime,
		}
		issuerParticipant2ID, err := k.CreateParticipant(sdkCtx, issuerParticipant2)
		require.NoError(t, err)

		// Expected: fees from validatorParticipant (100) * (1 - 0.3) = 70
		msg := &types.MsgCreateOrUpdateParticipantSession{
			Corporation:              authority,
			Operator:                 operator,
			Id:                       uuid.New().String(),
			IssuerParticipantId:      issuerParticipant2ID,
			VerifierParticipantId:    0,
			AgentParticipantId:       agentParticipantID,
			WalletAgentParticipantId: walletAgentParticipantID,
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
		validatorWithDiscountID, err := k.CreateParticipant(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		// Create issuer with discount (per Issue #94: use discount instead of exemption)
		issuerWithDiscount := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			VsOperator:             operator,
			Created:                &now,
			Adjusted:               &now,
			Modified:               &now,
			ValidatorParticipantId: validatorWithDiscountID,
			OpState:                types.OnboardingState_VALIDATED,
			IssuanceFeeDiscount:    3000, // 30% discount
			EffectiveFrom:          &pastTime,
		}
		issuerWithDiscountID, err := k.CreateParticipant(sdkCtx, issuerWithDiscount)
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
			AgentParticipantId:       agentParticipantID,
			WalletAgentParticipantId: walletAgentParticipantID,
		}

		resp, err := ms.CreateOrUpdateParticipantSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestGetParticipantByID tests the GetParticipantByID function
func TestGetParticipantByID(t *testing.T) {
	k, _, trkKeeper, _, ctx, _ := keepertest.ParticipantKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Create a test participant
	testParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}
	participantID, err := k.CreateParticipant(sdkCtx, testParticipant)
	require.NoError(t, err)

	// Test getting the participant
	retrievedParticipant, err := k.GetParticipantByID(sdkCtx, participantID)
	require.NoError(t, err, "GetParticipantByID should not return an error for a valid ID")
	require.Equal(t, participantID, retrievedParticipant.Id, "Participant ID should match")
	require.Equal(t, testParticipant.SchemaId, retrievedParticipant.SchemaId, "Schema ID should match")
	require.Equal(t, testParticipant.Role, retrievedParticipant.Role, "Type should match")
	require.Equal(t, testParticipant.CorporationId, retrievedParticipant.CorporationId, "Corporation should match")

	// Test getting a non-existent participant
	_, err = k.GetParticipantByID(sdkCtx, 9999)
	require.Error(t, err, "GetParticipantByID should return an error for an invalid ID")
}

// TestCreateAndUpdateParticipant tests the CreateParticipant and UpdateParticipant functions
func TestCreateAndUpdateParticipant(t *testing.T) {
	k, _, trkKeeper, _, ctx, _ := keepertest.ParticipantKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Test CreateParticipant
	testParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
	}

	participantID, err := k.CreateParticipant(sdkCtx, testParticipant)
	require.NoError(t, err, "CreateParticipant should not return an error")
	require.Greater(t, participantID, uint64(0), "Participant ID should be greater than 0")

	// Retrieve the created participant
	retrievedParticipant, err := k.GetParticipantByID(sdkCtx, participantID)
	require.NoError(t, err)
	require.Equal(t, participantID, retrievedParticipant.Id, "Created participant ID should match")
	require.Equal(t, testParticipant.SchemaId, retrievedParticipant.SchemaId, "Created participant schema ID should match")

	// Test UpdateParticipant
	futureTime := now.Add(24 * time.Hour)
	retrievedParticipant.EffectiveUntil = &futureTime

	err = k.UpdateParticipant(sdkCtx, retrievedParticipant)
	require.NoError(t, err, "UpdateParticipant should not return an error")

	// Retrieve the updated participant
	updatedParticipant, err := k.GetParticipantByID(sdkCtx, participantID)
	require.NoError(t, err)
	require.Equal(t, futureTime.Unix(), updatedParticipant.EffectiveUntil.Unix(), "EffectiveUntil should be updated")
}

// TestQueryParticipants tests the query functions for participants
func TestQueryParticipants(t *testing.T) {
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

	// Create several participants for testing
	// Trust Registry participant
	trustParticipant := types.Participant{
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
	trustParticipantID, err := k.CreateParticipant(sdkCtx, trustParticipant)
	require.NoError(t, err)

	// Issuer participant
	issuerParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		Did:                    validDid,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
	require.NoError(t, err)

	// Verifier participant
	verifierParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_VERIFIER,
		Did:                    validDid,
		CorporationId:          trkKeeper.RegisterCorp(creator),
		Created:                &now,
		Adjusted:               &now,
		Modified:               &now,
		ValidatorParticipantId: trustParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		EffectiveFrom:          &pastTime,
	}
	verifierParticipantID, err := k.CreateParticipant(sdkCtx, verifierParticipant)

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
				IssuerParticipantId:   issuerParticipantID,
				VerifierParticipantId: verifierParticipantID,
				AgentParticipantId:    issuerParticipantID, // Using issuer as agent for simplicity in test
			},
		},
	}
	err = k.ParticipantSession.Set(sdkCtx, sessionID, session)
	require.NoError(t, err)

	// Test GetParticipant query
	getParticipantReq := &types.QueryGetParticipantRequest{
		Id: issuerParticipantID,
	}
	getParticipantResp, err := k.GetParticipant(ctx, getParticipantReq)
	require.NoError(t, err)
	require.NotNil(t, getParticipantResp)
	require.Equal(t, issuerParticipantID, getParticipantResp.Participant.Id)
	require.Equal(t, validDid, getParticipantResp.Participant.Did)

	// Test ListParticipants query
	listParticipantReq := &types.QueryListParticipantsRequest{
		ResponseMaxSize: 10,
	}
	listParticipantResp, err := k.ListParticipants(ctx, listParticipantReq)
	require.NoError(t, err)
	require.NotNil(t, listParticipantResp)
	require.GreaterOrEqual(t, len(listParticipantResp.Participants), 3) // At least the 3 we created

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
	findParticipantDIDReq := &types.QueryFindParticipantsWithDIDRequest{
		Did:      validDid,
		Role:     uint32(types.ParticipantRole_ISSUER),
		SchemaId: 1,
	}
	findParticipantDIDResp, err := k.FindParticipantsWithDID(ctx, findParticipantDIDReq)
	require.NoError(t, err)
	require.NotNil(t, findParticipantDIDResp)
	require.Equal(t, 1, len(findParticipantDIDResp.Participants)) // Should find only the issuer participant
	require.Equal(t, issuerParticipantID, findParticipantDIDResp.Participants[0].Id)

	// Test FindBeneficiaries query
	findBenefReq := &types.QueryFindBeneficiariesRequest{
		IssuerParticipantId:   issuerParticipantID,
		VerifierParticipantId: verifierParticipantID,
	}
	findBenefResp, err := k.FindBeneficiaries(ctx, findBenefReq)
	require.NoError(t, err)
	require.NotNil(t, findBenefResp)
	require.GreaterOrEqual(t, len(findBenefResp.Participants), 1) // Should find the trust participant at minimum

	// Find the trust participant in the response
	foundTrustParticipant := false
	for _, participant := range findBenefResp.Participants {
		if participant.Id == trustParticipantID {
			foundTrustParticipant = true
			break
		}
	}
	require.True(t, foundTrustParticipant, "Trust registry participant should be in beneficiaries")
}

func TestSlashParticipantTrustDeposit(t *testing.T) {
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

	// Create validator participant (ISSUER_GRANTOR) owned by validatorAddr
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create applicant participant (ISSUER) with deposit, vs_operator set
	applicantParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                1000,
		EffectiveFrom:          &pastTime,
		VsOperator:             operator,
	}
	applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
	require.NoError(t, err)

	// Create a VERIFIER participant to test VS operator revocation
	verifierParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_VERIFIER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                500,
		EffectiveFrom:          &pastTime,
		VsOperator:             operator,
	}
	verifierParticipantID, err := k.CreateParticipant(sdkCtx, verifierParticipant)
	require.NoError(t, err)

	// Create an ECOSYSTEM participant (no VS operator revocation for non-ISSUER/VERIFIER)
	ecosystemParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(applicantAuthority),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		Deposit:       300,
		EffectiveFrom: &pastTime,
	}
	ecosystemParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	// Create expired participant (still slashable per spec)
	expiredTime := now.Add(-2 * time.Hour)
	expiredUntil := now.Add(-1 * time.Hour)
	expiredParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &expiredTime,
		Modified:               &expiredTime,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                200,
		EffectiveFrom:          &expiredTime,
		EffectiveUntil:         &expiredUntil,
	}
	expiredParticipantID, err := k.CreateParticipant(sdkCtx, expiredParticipant)
	require.NoError(t, err)

	// Create revoked participant (still slashable per spec)
	revokedParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAuthority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                200,
		EffectiveFrom:          &pastTime,
		Revoked:                &now,
	}
	revokedParticipantID, err := k.CreateParticipant(sdkCtx, revokedParticipant)
	require.NoError(t, err)

	t.Run("AUTHZ check - operator authorization failure", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          applicantParticipantID,
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
			Id:          applicantParticipantID,
			Amount:      100,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
		require.NoError(t, err)
		require.NotNil(t, participant.Slashed)
		require.Equal(t, uint64(100), participant.SlashedDeposit)
	})

	t.Run("Valid slash by TR controller", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: trControllerAddr,
			Operator:    operator,
			Id:          applicantParticipantID,
			Amount:      100,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
		require.NoError(t, err)
		require.Equal(t, uint64(200), participant.SlashedDeposit) // cumulative: 100 + 100
	})

	t.Run("Valid slash on expired participant (still slashable per spec)", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          expiredParticipantID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, expiredParticipantID)
		require.NoError(t, err)
		require.NotNil(t, participant.Slashed)
		require.Equal(t, uint64(50), participant.SlashedDeposit)
	})

	t.Run("Valid slash on revoked participant (still slashable per spec)", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          revokedParticipantID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, revokedParticipantID)
		require.NoError(t, err)
		require.NotNil(t, participant.Slashed)
	})

	t.Run("VS operator revocation on VERIFIER participant", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          verifierParticipantID,
			Amount:      50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("No VS operator revocation on ECOSYSTEM participant", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: trControllerAddr,
			Operator:    operator,
			Id:          ecosystemParticipantID,
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
		require.Contains(t, err.Error(), "participant not found")
		require.Nil(t, resp)
	})

	t.Run("Amount exceeds deposit", func(t *testing.T) {
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    operator,
			Id:          applicantParticipantID,
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
			Id:          applicantParticipantID,
			Amount:      10,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not authorized to slash this participant")
		require.Nil(t, resp)
	})

	t.Run("Wrong authority - applicant own authority cannot slash", func(t *testing.T) {
		// Unlike revoke, slash does NOT have Option #3 (self-authority)
		resp, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: applicantAuthority,
			Operator:    operator,
			Id:          applicantParticipantID,
			Amount:      10,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not authorized to slash this participant")
		require.Nil(t, resp)
	})

	_ = authority // suppress unused
}

func TestRepayParticipantSlashedTrustDeposit(t *testing.T) {
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

	// Create ecosystem participant
	ecosystemParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	// Create validator participant
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(validatorAddr),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// Create applicant participant owned by authority with initial deposit
	applicantParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                1000,
		EffectiveFrom:          &pastTime,
	}
	applicantParticipantID, err := k.CreateParticipant(sdkCtx, applicantParticipant)
	require.NoError(t, err)

	// Create unslashed participant (for negative test)
	unslashedParticipant := types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(authority),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorParticipantID,
		OpState:                types.OnboardingState_VALIDATED,
		Deposit:                500,
		EffectiveFrom:          &pastTime,
	}
	unslashedParticipantID, err := k.CreateParticipant(sdkCtx, unslashedParticipant)
	require.NoError(t, err)

	// Slash the applicant participant first
	slashMsg := &types.MsgSlashParticipantTrustDeposit{
		Corporation: validatorAddr,
		Operator:    validatorAddr,
		Id:          applicantParticipantID,
		Amount:      500,
	}
	_, err = ms.SlashParticipantTrustDeposit(ctx, slashMsg)
	require.NoError(t, err)

	// Verify slashed state
	slashedParticipant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
	require.NoError(t, err)
	require.Equal(t, uint64(500), slashedParticipant.SlashedDeposit)

	t.Run("AUTHZ check - operator authorization failure", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          applicantParticipantID,
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
			Id:          applicantParticipantID,
			Amount:      500,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify participant was updated correctly
		participant, err := k.GetParticipantByID(sdkCtx, applicantParticipantID)
		require.NoError(t, err)
		require.NotNil(t, participant.Repaid)
		require.NotNil(t, participant.Modified)
		require.Equal(t, uint64(500), participant.RepaidDeposit)
	})

	t.Run("Invalid - already fully repaid", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          applicantParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "slashed deposit already fully repaid")
		require.Nil(t, resp)
	})

	t.Run("Invalid - participant not found", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          9999,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "participant not found")
		require.Nil(t, resp)
	})

	t.Run("Invalid - wrong authority (not owner)", func(t *testing.T) {
		// Slash a new participant for this test
		newParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(otherAuthority),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			Deposit:                300,
			EffectiveFrom:          &pastTime,
		}
		otherParticipantID, err := k.CreateParticipant(sdkCtx, newParticipant)
		require.NoError(t, err)
		// Slash it
		_, err = ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    validatorAddr,
			Id:          otherParticipantID,
			Amount:      100,
		})
		require.NoError(t, err)

		// Try to repay with wrong authority
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority, // wrong - participant belongs to otherAuthority
			Operator:    operator,
			Id:          otherParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority is not the owner of this participant")
		require.Nil(t, resp)
	})

	t.Run("Invalid - no slashed deposit to repay", func(t *testing.T) {
		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          unslashedParticipantID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no slashed timestamp")
		require.Nil(t, resp)
	})

	t.Run("Invalid - amount exceeds outstanding slashed deposit", func(t *testing.T) {
		// Fresh participant slashed by 400; repaying 500 exceeds the outstanding 400.
		p := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: validatorParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			Deposit:                1000,
			EffectiveFrom:          &pastTime,
		}
		pid, err := k.CreateParticipant(sdkCtx, p)
		require.NoError(t, err)

		_, err = ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{
			Corporation: validatorAddr,
			Operator:    validatorAddr,
			Id:          pid,
			Amount:      400,
		})
		require.NoError(t, err)

		resp, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{
			Corporation: authority,
			Operator:    operator,
			Id:          pid,
			Amount:      500,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds outstanding")
		require.Nil(t, resp)
	})
}

func TestCreateParticipant(t *testing.T) {
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

	// Create ecosystem participant (active, with effective_until)
	ecosystemParticipant := types.Participant{
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
	ecosystemParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
	require.NoError(t, err)

	// Create ecosystem participant without effective_until (never expires)
	neverExpireParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ECOSYSTEM,
		Did:           validDid,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	neverExpireParticipantID, err := k.CreateParticipant(sdkCtx, neverExpireParticipant)
	require.NoError(t, err)

	t.Run("AUTHZ check - operator authorization failure", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator authorization not found")
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemParticipantID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Nil(t, resp)
		delKeeper.ErrToReturn = nil
	})

	t.Run("Valid ISSUER participant", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemParticipantID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
			VerificationFees:       100,
			ValidationFees:         50,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_ISSUER, participant.Role)
		require.NotZero(t, participant.CorporationId)
		require.Equal(t, validDid, participant.Did)
		require.Equal(t, ecosystemParticipantID, participant.ValidatorParticipantId)
		require.Equal(t, uint64(1), participant.SchemaId) // inherited from validator_participant
		require.Equal(t, uint64(100), participant.VerificationFees)
		require.Equal(t, uint64(50), participant.ValidationFees)
		require.Equal(t, uint64(0), participant.IssuanceFees)
		require.Equal(t, uint64(0), participant.Deposit)
		require.NotNil(t, participant.Created)
		require.NotNil(t, participant.Modified)
	})

	t.Run("Valid VERIFIER participant", func(t *testing.T) {
		futureTime2 := futureTime.Add(1 * time.Hour)
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: ecosystemParticipantID,
			Did:                    "did:example:verifier1",
			EffectiveFrom:          &futureTime2,
			EffectiveUntil:         &farFuture,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, resp.Id)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_VERIFIER, participant.Role)
		require.Equal(t, uint64(0), participant.VerificationFees)
		require.Equal(t, uint64(0), participant.ValidationFees)
	})

	t.Run("Invalid - validator participant not found", func(t *testing.T) {
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
		require.Contains(t, err.Error(), "validator participant not found")
		require.Nil(t, resp)
	})

	t.Run("Invalid - validator participant not ECOSYSTEM", func(t *testing.T) {
		// Create a non-ecosystem participant
		issuerParticipant := types.Participant{
			SchemaId:               1,
			Role:                   types.ParticipantRole_ISSUER,
			CorporationId:          trkKeeper.RegisterCorp(authority),
			Created:                &now,
			Modified:               &now,
			ValidatorParticipantId: ecosystemParticipantID,
			OpState:                types.OnboardingState_VALIDATED,
			EffectiveFrom:          &pastTime,
		}
		issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
		require.NoError(t, err)

		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: issuerParticipantID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &farFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "ECOSYSTEM participant")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_from not in future", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemParticipantID,
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
			ValidatorParticipantId: ecosystemParticipantID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &beforeFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be greater than effective_from")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_until exceeds validator_participant", func(t *testing.T) {
		wayFuture := farFuture.Add(24 * time.Hour)
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemParticipantID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			EffectiveUntil:         &wayFuture,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be <= validator_participant.effective_until")
		require.Nil(t, resp)
	})

	t.Run("Invalid - effective_until null but validator_participant has effective_until", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecosystemParticipantID,
			Did:                    validDid,
			EffectiveFrom:          &futureTime,
			// EffectiveUntil nil
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be set when validator_participant has effective_until")
		require.Nil(t, resp)
	})

	t.Run("Valid - both effective_until null when validator_participant never expires", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            otherAuthority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: neverExpireParticipantID,
			Did:                    "did:example:neverexpire",
			EffectiveFrom:          &futureTime,
			// EffectiveUntil nil - OK because validator_participant also has nil
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Invalid - VERIFIER with validation_fees", func(t *testing.T) {
		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: ecosystemParticipantID,
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

		// Create ecosystem participant for schema 2
		ecoParticipantS2 := types.Participant{
			SchemaId:       2,
			Role:           types.ParticipantRole_ECOSYSTEM,
			CorporationId:  trkKeeper.RegisterCorp(authority),
			Created:        &now,
			Modified:       &now,
			OpState:        types.OnboardingState_VALIDATED,
			EffectiveFrom:  &pastTime,
			EffectiveUntil: &farFuture,
		}
		ecoParticipantS2ID, err := k.CreateParticipant(sdkCtx, ecoParticipantS2)
		require.NoError(t, err)

		resp, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation:            authority,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: ecoParticipantS2ID,
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
// and it must be in the future. Per spec [MOD-PP-MSG-7-2-1]:
// - effective_from is mandatory
// - effective_from must be in the future

func TestCreateRootParticipant(t *testing.T) {
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
		// === Basic checks [MOD-PP-MSG-7-2-1] ===
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
		// === Participant checks [MOD-PP-MSG-7-2-2] ===
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
		// === Happy path [MOD-PP-MSG-7-3] ===
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

				// [MOD-PP-MSG-7-3] verify created participant per spec v4 draft 13:
				// participant.type is hardcoded to ECOSYSTEM, and participant.vs_operator is not set by this message.
				participant, err := k.GetParticipantByID(sdkCtx, resp.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.SchemaId, participant.SchemaId)
				require.Equal(t, types.ParticipantRole_ECOSYSTEM, participant.Role,
					"Create Root Participant MUST hardcode participant.type to ECOSYSTEM per spec [MOD-PP-MSG-7-3]")
				require.Empty(t, participant.VsOperator,
					"Create Root Participant MUST NOT set participant.vs_operator per spec [MOD-PP-MSG-7-3]")
				require.Equal(t, tc.msg.Did, participant.Did)
				require.NotZero(t, participant.CorporationId)
				require.Equal(t, now, *participant.Created)
				require.Equal(t, now, *participant.Modified)
				require.Equal(t, tc.msg.EffectiveFrom.Unix(), participant.EffectiveFrom.Unix())
				if tc.msg.EffectiveUntil != nil {
					require.Equal(t, tc.msg.EffectiveUntil.Unix(), participant.EffectiveUntil.Unix())
				} else {
					require.Nil(t, participant.EffectiveUntil)
				}
				require.Equal(t, tc.msg.ValidationFees, participant.ValidationFees)
				require.Equal(t, tc.msg.IssuanceFees, participant.IssuanceFees)
				require.Equal(t, tc.msg.VerificationFees, participant.VerificationFees)
				require.Equal(t, uint64(0), participant.Deposit)
			}
		})
	}
}

func TestCreateRootParticipant_OverlapChecks(t *testing.T) {
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

	// Create an existing participant: effective_from=+1h, effective_until=+24h
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

	t.Run("3. Overlap: existing participant with nil effective_until (never expires)", func(t *testing.T) {
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

	t.Run("4. Revoked/slashed/repaid participants excluded from overlap", func(t *testing.T) {
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

		// Mark the participant as revoked
		participant, err := k.GetParticipantByID(sdkCtx, resp3.Id)
		require.NoError(t, err)
		revokedTime := now
		participant.Revoked = &revokedTime
		err = k.Participant.Set(sdkCtx, participant.Id, participant)
		require.NoError(t, err)

		// Now create a new participant that would overlap if the revoked one was active → should succeed
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

	t.Run("5. No overlap: new participant starts after existing ends", func(t *testing.T) {
		// Use schema 1 with existing participant: +1h to +24h
		// But existing.effective_from < new.effective_until still causes overlap
		// To truly avoid overlap, need participant on a different schema OR existing must be expired/revoked
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

		// New participant starts after first ends: +6h to +10h
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

func TestCreateRootParticipant_AuthzCheck(t *testing.T) {
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
// ISSUE #193: StartParticipantOP - Validator participant must be ACTIVE
// =============================================================================
// This test validates that StartParticipantOP requires the validator participant
// to be ACTIVE (not INACTIVE, REVOKED, EXPIRED, etc). Per spec:
// - validator_participant must be a valid participant
// - If effective_from is null or in the future, participant is INACTIVE/FUTURE
// - If revoked, slashed, or expired, participant is invalid

func TestStartParticipantVP_ValidatorMustBeActive(t *testing.T) {
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
	pastTime := now.Add(-1 * time.Hour)     // In the past - for ACTIVE participants
	futureTime := now.Add(1 * time.Hour)    // In the future - for FUTURE/INACTIVE participants
	expiredTime := now.Add(-24 * time.Hour) // Far in the past - for EXPIRED participants

	// Create an ACTIVE validator participant (valid case for comparison)
	activeValidatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // In the past = ACTIVE
	}
	activeValidatorParticipantID, err := k.CreateParticipant(sdkCtx, activeValidatorParticipant)
	require.NoError(t, err)

	// Issue #193: Create a validator participant with NO effective_from (INACTIVE)
	inactiveValidatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: nil, // NULL effective_from = INACTIVE
	}
	inactiveValidatorParticipantID, err := k.CreateParticipant(sdkCtx, inactiveValidatorParticipant)
	require.NoError(t, err)

	// Issue #193: Create a validator participant with FUTURE effective_from
	futureValidatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &futureTime, // Future effective_from = not yet ACTIVE
	}
	futureValidatorParticipantID, err := k.CreateParticipant(sdkCtx, futureValidatorParticipant)
	require.NoError(t, err)

	// Issue #193: Create an EXPIRED validator participant
	expiredValidatorParticipant := types.Participant{
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
	expiredValidatorParticipantID, err := k.CreateParticipant(sdkCtx, expiredValidatorParticipant)
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
				ValidatorParticipantId: activeValidatorParticipantID,
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
				ValidatorParticipantId: inactiveValidatorParticipantID,
				Did:                    validDid,
			},
			expectErr: true,
			errMsg:    "validator participant is not valid",
		},
		{
			// Issue #193: Validator with future effective_from should be rejected
			name: "Issue #193: Reject FUTURE validator - effective_from is in the future",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: futureValidatorParticipantID,
				Did:                    validDid,
			},
			expectErr: true,
			errMsg:    "validator participant is not valid",
		},
		{
			// Issue #193: Expired validator should be rejected
			name: "Issue #193: Reject EXPIRED validator - effective_until has passed",
			msg: &types.MsgStartParticipantOP{
				Corporation:            creator,
				Operator:               creator,
				Role:                   types.ParticipantRole_ISSUER,
				ValidatorParticipantId: expiredValidatorParticipantID,
				Did:                    validDid,
			},
			expectErr: true,
			errMsg:    "validator participant is not valid",
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
// ISSUE #196: RevokeParticipant - Allow revoking not-yet-active participants
// =============================================================================
// This test validates that RevokeParticipant allows revoking participants that
// are not yet active (e.g., effective_from is in the future or null).
// Per spec, no IsValidParticipant check is required for revocation.

// TestRevokeParticipant_RequiresActiveParticipant tests that v4 spec requires
// applicant_participant to be an active participant (reverting Issue #196 relaxation).
func TestRevokeParticipant_RequiresActiveParticipant(t *testing.T) {
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

	// Create an ACTIVE participant (for comparison)
	activeParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime, // ACTIVE
	}
	activeParticipantID, err := k.CreateParticipant(sdkCtx, activeParticipant)
	require.NoError(t, err)

	// Create a participant with FUTURE effective_from (not yet active)
	futureParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &futureTime, // FUTURE - not yet active
	}
	futureParticipantID, err := k.CreateParticipant(sdkCtx, futureParticipant)
	require.NoError(t, err)

	// Create a participant with NULL effective_from (inactive)
	inactiveParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(authority),
		Created:       &now,
		Adjusted:      &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: nil, // INACTIVE - no effective_from
	}
	inactiveParticipantID, err := k.CreateParticipant(sdkCtx, inactiveParticipant)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		msg       *types.MsgRevokeParticipant
		expectErr bool
		errMsg    string
	}{
		{
			// Baseline: Revoking an ACTIVE participant should work
			name: "Revoke ACTIVE participant - valid case",
			msg: &types.MsgRevokeParticipant{
				Corporation: authority,
				Operator:    operatorAddr,
				Id:          activeParticipantID,
			},
			expectErr: false,
			errMsg:    "",
		},
		{
			// v4 spec: FUTURE participant (not yet active) should be rejected
			name: "Revoke FUTURE participant - not yet active should be rejected",
			msg: &types.MsgRevokeParticipant{
				Corporation: authority,
				Operator:    operatorAddr,
				Id:          futureParticipantID,
			},
			expectErr: true,
			errMsg:    "applicant participant is not active",
		},
		{
			// v4 spec: INACTIVE participant (null effective_from) should be rejected
			name: "Revoke INACTIVE participant - null effective_from should be rejected",
			msg: &types.MsgRevokeParticipant{
				Corporation: authority,
				Operator:    operatorAddr,
				Id:          inactiveParticipantID,
			},
			expectErr: true,
			errMsg:    "applicant participant is not active",
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

				// Verify the participant was revoked
				participant, err := k.GetParticipantByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, participant.Revoked, "Participant should be revoked")
			}
		})
	}
}

// TestStartParticipantVP_OverlapCheck tests [MOD-PP-MSG-1-2-4]:
// Cannot have 2 active VPs in the same (schema_id, type, validator_participant_id, authority) context.
func TestStartParticipantVP_OverlapCheck(t *testing.T) {
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
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	// First VP should succeed
	msg := &types.MsgStartParticipantOP{
		Corporation:            creator,
		Operator:               creator,
		Role:                   types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorParticipantID,
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
			ValidatorParticipantId: validatorParticipantID,
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
			ValidatorParticipantId: validatorParticipantID,
			Did:                    "did:example:overlap-other-authority",
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

		verifierGrantorParticipant := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgParticipantID, err := k.CreateParticipant(sdkCtx, verifierGrantorParticipant)
		require.NoError(t, err)

		msg4 := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               creator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: vgParticipantID,
			Did:                    validDid,
		}
		resp4, err := ms.StartParticipantOP(ctx, msg4)
		require.NoError(t, err)
		require.NotNil(t, resp4)
	})
}

// TestStartParticipantVP_AuthzCheck tests that the AUTHZ-CHECK via DelegationKeeper
// is properly enforced when the keeper is present.
func TestStartParticipantVP_AuthzCheck(t *testing.T) {
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
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreateParticipant(sdkCtx, validatorParticipant)
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

// TestStartParticipantVP_VsOperatorAndFields tests that vs_operator fields and DID are correctly
// persisted, and that empty DID is rejected at the keeper level.
func TestStartParticipantVP_VsOperatorAndFields(t *testing.T) {
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
	validatorParticipant := types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(creator),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorParticipantID, err := k.CreateParticipant(sdkCtx, validatorParticipant)
	require.NoError(t, err)

	t.Run("vs_operator fields propagated to stored participant", func(t *testing.T) {
		operator := sdk.AccAddress([]byte("diff_operator_aa")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            creator,
			Operator:               operator,
			Role:                   types.ParticipantRole_ISSUER,
			ValidatorParticipantId: validatorParticipantID,
			Did:                    validDid,
			VsOperator:             vsOperator,
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, validDid, participant.Did, "DID should be stored")
		require.NotZero(t, participant.CorporationId, "Corporation should be set")
		require.Equal(t, vsOperator, participant.VsOperator, "VsOperator should be stored")
		require.Equal(t, uint64(1), participant.SchemaId, "SchemaId should be derived from validator perm")
		require.Equal(t, types.OnboardingState_PENDING, participant.OpState)
	})

	t.Run("VERIFIER with VERIFIER_GRANTOR validator", func(t *testing.T) {
		vgParticipant := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_VERIFIER_GRANTOR,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgParticipantID, err := k.CreateParticipant(sdkCtx, vgParticipant)
		require.NoError(t, err)

		verifierCreator := sdk.AccAddress([]byte("verifier_creator")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            verifierCreator,
			Operator:               verifierCreator,
			Role:                   types.ParticipantRole_VERIFIER,
			ValidatorParticipantId: vgParticipantID,
			Did:                    "did:example:verifier-did-123",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_VERIFIER, participant.Role)
		require.Equal(t, vgParticipantID, participant.ValidatorParticipantId)
	})

	t.Run("HOLDER with ISSUER validator", func(t *testing.T) {
		// Create ISSUER participant to serve as validator for HOLDER
		issuerParticipant := types.Participant{
			SchemaId:      1,
			Role:          types.ParticipantRole_ISSUER,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		issuerParticipantID, err := k.CreateParticipant(sdkCtx, issuerParticipant)
		require.NoError(t, err)

		holderCreator := sdk.AccAddress([]byte("holder_creator_a")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            holderCreator,
			Operator:               holderCreator,
			Role:                   types.ParticipantRole_HOLDER,
			ValidatorParticipantId: issuerParticipantID,
			Did:                    "did:example:holder-did-456",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_HOLDER, participant.Role)
		require.Equal(t, issuerParticipantID, participant.ValidatorParticipantId)
	})

	t.Run("HOLDER with wrong validator type rejects", func(t *testing.T) {
		holderCreator := sdk.AccAddress([]byte("holder_bad_val_a")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            holderCreator,
			Operator:               holderCreator,
			Role:                   types.ParticipantRole_HOLDER,
			ValidatorParticipantId: validatorParticipantID, // ISSUER_GRANTOR, not ISSUER
			Did:                    "did:example:holder-bad-val",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "holder participant requires ISSUER validator")
		require.Nil(t, resp)
	})

	t.Run("ECOSYSTEM type combination - ISSUER_GRANTOR with ECOSYSTEM validator", func(t *testing.T) {
		// Create schema with ECOSYSTEM mode for issuer
		csKeeper.UpdateMockCredentialSchema(2, trID,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)

		ecosystemParticipant := types.Participant{
			SchemaId:      2,
			Role:          types.ParticipantRole_ECOSYSTEM,
			CorporationId: trkKeeper.RegisterCorp(creator),
			Created:       &now,
			Modified:      &now,
			OpState:       types.OnboardingState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		ecoParticipantID, err := k.CreateParticipant(sdkCtx, ecosystemParticipant)
		require.NoError(t, err)

		grantorCreator := sdk.AccAddress([]byte("grantor_eco_crea")).String()
		msg := &types.MsgStartParticipantOP{
			Corporation:            grantorCreator,
			Operator:               grantorCreator,
			Role:                   types.ParticipantRole_ISSUER_GRANTOR,
			ValidatorParticipantId: ecoParticipantID,
			Did:                    "did:example:issuer-grantor-eco",
		}
		resp, err := ms.StartParticipantOP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		participant, err := k.GetParticipantByID(sdkCtx, resp.ParticipantId)
		require.NoError(t, err)
		require.Equal(t, types.ParticipantRole_ISSUER_GRANTOR, participant.Role)
	})
}

// =============================================================================
// VSOA wiring (spec v4-rc2): MSG-1 creates a disabled record via MOD-DE-MSG-5;
// MSG-3 activates it via MOD-DE-MSG-9. These assert the new DelegationKeeper.
// =============================================================================

func vsoaValidator(t *testing.T, k keeper.Keeper, sdkCtx sdk.Context, trkKeeper interface{ RegisterCorp(string) uint64 }, corp string, now, past time.Time) uint64 {
	t.Helper()
	id, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId:      1,
		Role:          types.ParticipantRole_ISSUER_GRANTOR,
		CorporationId: trkKeeper.RegisterCorp(corp),
		Created:       &now,
		Modified:      &now,
		OpState:       types.OnboardingState_VALIDATED,
		EffectiveFrom: &past,
	})
	require.NoError(t, err)
	return id
}

func TestVSOA_StartParticipantOPGrantsDisabledRecord(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)

	creator := sdk.AccAddress([]byte("vsoa_creator________")).String()
	vsOperator := sdk.AccAddress([]byte("vsoa_vs_operator____")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	validatorPermID := vsoaValidator(t, k, sdkCtx, trkKeeper, creator, now, past)
	delKeeper.Reset()

	resp, err := ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation:             creator,
		Operator:                creator,
		Role:                    types.ParticipantRole_ISSUER,
		ValidatorParticipantId:  validatorPermID,
		Did:                     "did:example:vsoa-issuer",
		VsOperator:              vsOperator,
		VsOperatorAuthzMsgTypes: []string{types.MsgCreateOrUpdateParticipantSessionTypeURL},
	})
	require.NoError(t, err)

	require.Len(t, delKeeper.GrantVSOACalls, 1)
	require.Equal(t, vsOperator, delKeeper.GrantVSOACalls[0].VsOperator)
	require.Equal(t, resp.ParticipantId, delKeeper.GrantVSOACalls[0].Record.ParticipantId)
	require.NotNil(t, delKeeper.GrantVSOACalls[0].Record.Expiration)
	require.True(t, delKeeper.GrantVSOACalls[0].Record.Expiration.Equal(now), "record created disabled (expiration == now)")
}

func TestVSOA_StartParticipantOPSkipsWithoutMsgTypes(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)

	creator := sdk.AccAddress([]byte("vsoa_creator2_______")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	validatorPermID := vsoaValidator(t, k, sdkCtx, trkKeeper, creator, now, past)
	delKeeper.Reset()

	_, err := ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{
		Corporation:            creator,
		Operator:               creator,
		Role:                   types.ParticipantRole_ISSUER,
		ValidatorParticipantId: validatorPermID,
		Did:                    "did:example:vsoa-issuer2",
	})
	require.NoError(t, err)
	require.Len(t, delKeeper.GrantVSOACalls, 0)
}

func TestVSOA_ValidatedUpdatesExpiration(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx, delKeeper := setupMsgServerWithDelegation(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	ctx = sdk.WrapSDKContext(sdkCtx)
	now := sdkCtx.BlockTime()
	past := now.Add(-1 * time.Hour)
	future := now.Add(365 * 24 * time.Hour)

	validatorAddr := sdk.AccAddress([]byte("vsoa_validator______")).String()
	applicantAddr := sdk.AccAddress([]byte("vsoa_applicant______")).String()
	vsOperator := sdk.AccAddress([]byte("vsoa_vsop___________")).String()
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS)
	validatorPermID := vsoaValidator(t, k, sdkCtx, trkKeeper, validatorAddr, now, past)
	applicantPermID, err := k.CreateParticipant(sdkCtx, types.Participant{
		SchemaId:               1,
		Role:                   types.ParticipantRole_ISSUER,
		CorporationId:          trkKeeper.RegisterCorp(applicantAddr),
		Created:                &now,
		Modified:               &now,
		ValidatorParticipantId: validatorPermID,
		OpState:                types.OnboardingState_PENDING,
		VsOperator:             vsOperator,
	})
	require.NoError(t, err)
	delKeeper.Reset()

	_, err = ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{
		Corporation:      validatorAddr,
		Operator:         validatorAddr,
		Id:               applicantPermID,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveUntil:   &future,
		OpSummaryDigest:  "sha384-validDigest",
	})
	require.NoError(t, err)

	require.Len(t, delKeeper.UpdateVSOACalls, 1)
	require.Equal(t, applicantPermID, delKeeper.UpdateVSOACalls[0].ParticipantID)
	require.True(t, delKeeper.UpdateVSOACalls[0].NewExpiration.Equal(future))
}
