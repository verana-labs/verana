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
	"github.com/verana-labs/verana/x/perm/keeper"
	"github.com/verana-labs/verana/x/perm/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockCredentialSchemaKeeper, *keepertest.MockTrustRegistryKeeper, context.Context) {
	k, csKeeper, trkKeeper, ctx, _ := keepertest.PermissionKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx
}

func setupMsgServerWithDelegation(t testing.TB) (keeper.Keeper, types.MsgServer, *keepertest.MockCredentialSchemaKeeper, *keepertest.MockTrustRegistryKeeper, context.Context, *keepertest.MockDelegationKeeper) {
	k, csKeeper, trkKeeper, ctx, delKeeper := keepertest.PermissionKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx, delKeeper
}

func TestMsgServer(t *testing.T) {
	k, ms, _, _, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

// Test for StartPermissionVP
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
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema with specific perm management modes
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Create validator perm (ISSUER_GRANTOR)
	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE
	// This should be VALIDATED as it's a prerequisite
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED, // validator must be validated
		EffectiveFrom: &pastTime,                       // Required for ACTIVE state
	}

	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create another validator perm (VERIFIER_GRANTOR with different country)
	verifierGrantorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_VERIFIER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "FR", // Different country
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	verifierGrantorPermID, err := k.CreatePermission(sdkCtx, verifierGrantorPerm)
	require.NoError(t, err)

	// Create a validator perm without country (for testing optional country)
	validatorPermNoCountry := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "", // No country restriction
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	validatorPermNoCountryID, err := k.CreatePermission(sdkCtx, validatorPermNoCountry)
	require.NoError(t, err)

	testCases := []struct {
		name                     string
		msg                      *types.MsgStartPermissionVP
		err                      string
		checkFees                bool
		expectedValidationFees   uint64
		expectedIssuanceFees     uint64
		expectedVerificationFees uint64
	}{
		{
			name: "Valid ISSUER Permission Request",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: validatorPermID,
				Did:             validDid,
			},
			err:       "",
			checkFees: false,
		},
		{
			name: "Valid ISSUER Permission Request with optional fees",
			msg: &types.MsgStartPermissionVP{
				Authority:        creator2,
				Operator:         creator2,
				Type:             types.PermissionType_ISSUER,
				ValidatorPermId:  validatorPermID,
				Did:              validDid,
				ValidationFees:   &types.OptionalUInt64{Value: 100},
				IssuanceFees:     &types.OptionalUInt64{Value: 50},
				VerificationFees: &types.OptionalUInt64{Value: 25},
			},
			err:                      "",
			checkFees:                true,
			expectedValidationFees:   100,
			expectedIssuanceFees:     50,
			expectedVerificationFees: 25,
		},
		{
			name: "Valid ISSUER Permission Request with partial fees",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator3,
				Operator:        creator3,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: validatorPermID,
				Did:             validDid,
				ValidationFees:  &types.OptionalUInt64{Value: 75},
			},
			err:                      "",
			checkFees:                true,
			expectedValidationFees:   75,
			expectedIssuanceFees:     0,
			expectedVerificationFees: 0,
		},
		{
			name: "Valid ISSUER Permission Request with zero fees",
			msg: &types.MsgStartPermissionVP{
				Authority:        creator4,
				Operator:         creator4,
				Type:             types.PermissionType_ISSUER,
				ValidatorPermId:  validatorPermID,
				Did:              validDid,
				ValidationFees:   &types.OptionalUInt64{Value: 0},
				IssuanceFees:     &types.OptionalUInt64{Value: 0},
				VerificationFees: &types.OptionalUInt64{Value: 0},
			},
			err:                      "",
			checkFees:                true,
			expectedValidationFees:   0,
			expectedIssuanceFees:     0,
			expectedVerificationFees: 0,
		},
		{
			name: "Valid ISSUER Permission Request without country on validator",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: validatorPermNoCountryID,
				Did:             validDid,
			},
			err:       "",
			checkFees: false,
		},
		{
			name: "Non-existent Validator Permission",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: 999,
				Did:             validDid,
			},
			err:       "validator perm not found",
			checkFees: false,
		},
		{
			name: "Invalid Permission Type Combination - ISSUER with wrong validator",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: verifierGrantorPermID, // Wrong validator type
				Did:             validDid,
			},
			err:       "issuer perm requires ISSUER_GRANTOR validator",
			checkFees: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.StartPermissionVP(ctx, tc.msg)
			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Greater(t, resp.PermissionId, uint64(0))

				// Verify created perm
				perm, err := k.GetPermissionByID(sdkCtx, resp.PermissionId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Type, perm.Type)
				require.Equal(t, tc.msg.Authority, perm.Authority)
				require.Equal(t, tc.msg.ValidatorPermId, perm.ValidatorPermId)
				require.Equal(t, types.ValidationState_PENDING, perm.VpState)
				require.NotNil(t, perm.Created)
				require.NotNil(t, perm.Modified)
				require.NotNil(t, perm.VpLastStateChange)

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
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Create validator perm
	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          3, // ISSUER_GRANTOR
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}

	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)

	require.NoError(t, err)

	// Create applicant perm
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            1, // ISSUER
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	applicantPermID, err := k.CreatePermission(sdk.UnwrapSDKContext(ctx), applicantPerm)
	require.NoError(t, err)

	testCases := []struct {
		name string
		msg  *types.MsgRenewPermissionVP
		err  string
	}{
		{
			name: "Non-existent Permission",
			msg: &types.MsgRenewPermissionVP{
				Authority: creator,
				Operator:  creator,
				Id:        999,
			},
			err: "perm not found",
		},
		{
			name: "Wrong Authority",
			msg: &types.MsgRenewPermissionVP{
				Authority: sdk.AccAddress([]byte("wrong_creator")).String(),
				Operator:  sdk.AccAddress([]byte("wrong_creator")).String(),
				Id:        applicantPermID,
			},
			err: "authority is not the perm authority",
		},
		{
			name: "Successful Renewal",
			msg: &types.MsgRenewPermissionVP{
				Authority: creator,
				Operator:  creator,
				Id:        applicantPermID,
			},
			err: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RenewPermissionVP(ctx, tc.msg)
			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify updated perm
				perm, err := k.GetPermissionByID(sdk.UnwrapSDKContext(ctx), tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, types.ValidationState_PENDING, perm.VpState)
				require.NotNil(t, perm.VpLastStateChange)
			}
		})
	}
}

func TestRenewPermissionVP_AuthzCheck(t *testing.T) {
	k, ms, csKeeper, _, ctx, mockDelegation := setupMsgServerWithDelegation(t)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          3,
		Authority:     creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            1,
		Authority:       creator,
		Created:         &now,
		CreatedBy:       creator,
		Modified:        &now,
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	t.Run("AUTHZ-CHECK failure blocks renewal", func(t *testing.T) {
		mockDelegation.ErrToReturn = fmt.Errorf("operator not authorized")
		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Nil(t, resp)
	})

	t.Run("AUTHZ-CHECK success allows renewal", func(t *testing.T) {
		mockDelegation.ErrToReturn = nil
		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, perm.VpState)
	})
}

func TestRenewPermissionVP_VpStatePrecondition(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          3,
		Authority:     creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("Renewing PENDING perm is blocked (prevents fee accounting loss)", func(t *testing.T) {
		pendingPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
			VpCurrentFees:   1000, // funds already in escrow
			VpCurrentDeposit: 500,
		}
		pendingPermID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        pendingPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "vp_state must be VALIDATED to renew")
		require.Nil(t, resp)

		// Verify the perm was NOT modified (fees still intact)
		perm, err := k.GetPermissionByID(sdkCtx, pendingPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, perm.VpState)
		require.Equal(t, uint64(1000), perm.VpCurrentFees)
		require.Equal(t, uint64(500), perm.VpCurrentDeposit)
	})

	t.Run("Renewing TERMINATED perm is blocked", func(t *testing.T) {
		terminatedPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_TERMINATED,
		}
		terminatedPermID, err := k.CreatePermission(sdkCtx, terminatedPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        terminatedPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "vp_state must be VALIDATED to renew")
		require.Nil(t, resp)
	})

	t.Run("Renewing TERMINATION_REQUESTED perm is blocked", func(t *testing.T) {
		termReqPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_TERMINATION_REQUESTED,
		}
		termReqPermID, err := k.CreatePermission(sdkCtx, termReqPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        termReqPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "vp_state must be VALIDATED to renew")
		require.Nil(t, resp)
	})

	t.Run("Renewing UNSPECIFIED vp_state perm is blocked", func(t *testing.T) {
		unspecPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATION_STATE_UNSPECIFIED,
		}
		unspecPermID, err := k.CreatePermission(sdkCtx, unspecPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        unspecPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "vp_state must be VALIDATED to renew")
		require.Nil(t, resp)
	})

	t.Run("Renewing VALIDATED perm succeeds", func(t *testing.T) {
		validatedPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED,
		}
		validatedPermID, err := k.CreatePermission(sdkCtx, validatedPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        validatedPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, validatedPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, perm.VpState)
		require.NotNil(t, perm.VpLastStateChange)
		require.Equal(t, now, *perm.VpLastStateChange)
		require.Equal(t, now, *perm.Modified)
	})
}

func TestRenewPermissionVP_ValidatorPermChecks(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	t.Run("Renewal blocked when validator perm is revoked", func(t *testing.T) {
		revokedTime := now.Add(-30 * time.Minute)
		revokedValidatorPerm := types.Permission{
			SchemaId:      1,
			Type:          3,
			Authority:     creator,
			Created:       &now,
			CreatedBy:     creator,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
			Revoked:       &revokedTime,
		}
		revokedValPermID, err := k.CreatePermission(sdkCtx, revokedValidatorPerm)
		require.NoError(t, err)

		applicantPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: revokedValPermID,
			VpState:         types.ValidationState_VALIDATED,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator perm is expired", func(t *testing.T) {
		expiredTime := now.Add(-10 * time.Minute)
		expiredValidatorPerm := types.Permission{
			SchemaId:       1,
			Type:           3,
			Authority:      creator,
			Created:        &now,
			CreatedBy:      creator,
			Modified:       &now,
			VpState:        types.ValidationState_VALIDATED,
			EffectiveFrom:  &pastTime,
			EffectiveUntil: &expiredTime,
		}
		expiredValPermID, err := k.CreatePermission(sdkCtx, expiredValidatorPerm)
		require.NoError(t, err)

		applicantPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: expiredValPermID,
			VpState:         types.ValidationState_VALIDATED,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator perm is INACTIVE (no effective_from)", func(t *testing.T) {
		inactiveValidatorPerm := types.Permission{
			SchemaId:  1,
			Type:      3,
			Authority: creator,
			Created:   &now,
			CreatedBy: creator,
			Modified:  &now,
			VpState:   types.ValidationState_VALIDATED,
			// EffectiveFrom is nil => INACTIVE
		}
		inactiveValPermID, err := k.CreatePermission(sdkCtx, inactiveValidatorPerm)
		require.NoError(t, err)

		applicantPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: inactiveValPermID,
			VpState:         types.ValidationState_VALIDATED,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	t.Run("Renewal blocked when validator perm does not exist", func(t *testing.T) {
		applicantPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: 99999, // non-existent
			VpState:         types.ValidationState_VALIDATED,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm not found")
		require.Nil(t, resp)
	})
}

func TestRenewPermissionVP_FeeAndDepositAccumulation(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// MockTrustRegistryKeeper returns trust_unit_price=1 by default
	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)

	validatorPerm := types.Permission{
		SchemaId:       1,
		Type:           3,
		Authority:      creator,
		Created:        &now,
		CreatedBy:      creator,
		Extended:       &now,
		ExtendedBy:     creator,
		Modified:       &now,
		VpState:        types.ValidationState_VALIDATED,
		EffectiveFrom:  &pastTime,
		ValidationFees: 50, // 50 trust units * 1 price = 50 denom fees
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("Deposit accumulates on renewal", func(t *testing.T) {
		initialDeposit := uint64(100)
		applicantPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED,
			Deposit:         initialDeposit,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  creator,
			Id:        applicantPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, perm.VpState)
		// Deposit should accumulate: initialDeposit + new deposit
		require.True(t, perm.Deposit >= initialDeposit, "deposit should accumulate, got %d", perm.Deposit)
		require.True(t, perm.VpCurrentFees > 0 || perm.VpCurrentDeposit > 0 || validatorPerm.ValidationFees == 0,
			"current fees or deposit should be set based on validator fees")
	})

	t.Run("Different operator than authority is allowed", func(t *testing.T) {
		operator := sdk.AccAddress([]byte("different_oper")).String()
		applicantPerm := types.Permission{
			SchemaId:        1,
			Type:            1,
			Authority:       creator,
			Created:         &now,
			CreatedBy:       creator,
			Modified:        &now,
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED,
		}
		applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
		require.NoError(t, err)

		resp, err := ms.RenewPermissionVP(ctx, &types.MsgRenewPermissionVP{
			Authority: creator,
			Operator:  operator,
			Id:        applicantPermID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, applicantPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_PENDING, perm.VpState)
	})
}

func TestRenewPermissionVP_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name string
		msg  *types.MsgRenewPermissionVP
		err  string
	}{
		{
			name: "Empty authority address",
			msg: &types.MsgRenewPermissionVP{
				Authority: "",
				Operator:  sdk.AccAddress([]byte("test_operator")).String(),
				Id:        1,
			},
			err: "invalid authority address",
		},
		{
			name: "Invalid authority address",
			msg: &types.MsgRenewPermissionVP{
				Authority: "invalid_address",
				Operator:  sdk.AccAddress([]byte("test_operator")).String(),
				Id:        1,
			},
			err: "invalid authority address",
		},
		{
			name: "Empty operator address",
			msg: &types.MsgRenewPermissionVP{
				Authority: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:  "",
				Id:        1,
			},
			err: "invalid operator address",
		},
		{
			name: "Invalid operator address",
			msg: &types.MsgRenewPermissionVP{
				Authority: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:  "invalid_address",
				Id:        1,
			},
			err: "invalid operator address",
		},
		{
			name: "Zero perm ID",
			msg: &types.MsgRenewPermissionVP{
				Authority: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:  sdk.AccAddress([]byte("test_operator")).String(),
				Id:        0,
			},
			err: "perm ID cannot be 0",
		},
		{
			name: "Valid message",
			msg: &types.MsgRenewPermissionVP{
				Authority: sdk.AccAddress([]byte("test_authority")).String(),
				Operator:  sdk.AccAddress([]byte("test_operator")).String(),
				Id:        1,
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
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
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
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()

	futureTime := now.Add(365 * 24 * time.Hour)
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       validatorAddr,
		Created:       &now,
		CreatedBy:     validatorAddr,
		Extended:      &now,
		ExtendedBy:    validatorAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}

	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// 1. Test with new perm (not renewal case)
	t.Run("Valid new perm validation", func(t *testing.T) {
		// Create a new perm in PENDING state
		newPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		// Set perm to validated
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      newPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     0, // Default no discount
			VerificationFeeDiscount: 0, // Default no discount
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify perm was updated correctly
		updatedPerm, err := k.GetPermissionByID(sdkCtx, newPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_VALIDATED, updatedPerm.VpState)
		require.Equal(t, msg.ValidationFees, updatedPerm.ValidationFees)
		require.Equal(t, msg.IssuanceFees, updatedPerm.IssuanceFees)
		require.Equal(t, msg.VerificationFees, updatedPerm.VerificationFees)
		require.Equal(t, msg.IssuanceFeeDiscount, updatedPerm.IssuanceFeeDiscount)
		require.Equal(t, msg.VerificationFeeDiscount, updatedPerm.VerificationFeeDiscount)
		require.NotNil(t, updatedPerm.EffectiveFrom)
		require.Equal(t, now.Unix(), updatedPerm.EffectiveFrom.Unix()) // First time: set to now
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, futureTime.Unix(), updatedPerm.EffectiveUntil.Unix())
		require.Equal(t, msg.VpSummaryDigestSri, updatedPerm.VpSummaryDigestSri)
		// Execution assertions
		require.NotNil(t, updatedPerm.Modified)
		require.Equal(t, now.Unix(), updatedPerm.Modified.Unix())
		require.NotNil(t, updatedPerm.VpLastStateChange)
		require.Equal(t, now.Unix(), updatedPerm.VpLastStateChange.Unix())
		require.Equal(t, uint64(0), updatedPerm.VpCurrentFees)    // Reset to 0
		require.Equal(t, uint64(0), updatedPerm.VpCurrentDeposit) // Reset to 0
	})

	// 2. Test renewal case - perm already has EffectiveFrom
	t.Run("Renewal perm validation", func(t *testing.T) {
		renewalAddr := sdk.AccAddress([]byte("renewal_creator")).String()
		// Create a perm that already has EffectiveFrom set (renewal)
		effectiveFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
		currentEffectiveUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Permission{
			SchemaId:         1,
			Type:             types.PermissionType_ISSUER,
			Authority:        renewalAddr,
			Created:          &now,
			CreatedBy:        renewalAddr,
			Extended:         &now,
			ExtendedBy:       renewalAddr,
			Modified:         &now,
			Country:          "US",
			ValidatorPermId:  validatorPermID,
			VpState:          types.ValidationState_PENDING,
			EffectiveFrom:    &effectiveFrom,
			EffectiveUntil:   &currentEffectiveUntil,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
		}
		renewalPermID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		// Set perm to validated with same fees
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      renewalPermID,
			ValidationFees:          10, // Same as existing
			IssuanceFees:            5,  // Same as existing
			VerificationFees:        3,  // Same as existing
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-renewalDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify perm was updated correctly
		updatedPerm, err := k.GetPermissionByID(sdkCtx, renewalPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_VALIDATED, updatedPerm.VpState)
		// Fees should remain unchanged (renewal doesn't overwrite)
		require.Equal(t, renewalPerm.ValidationFees, updatedPerm.ValidationFees)
		require.Equal(t, renewalPerm.IssuanceFees, updatedPerm.IssuanceFees)
		require.Equal(t, renewalPerm.VerificationFees, updatedPerm.VerificationFees)
		// EffectiveFrom should NOT change on renewal
		require.Equal(t, effectiveFrom.Unix(), updatedPerm.EffectiveFrom.Unix())
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, futureTime.Unix(), updatedPerm.EffectiveUntil.Unix())
	})

	// 3. Test validation error - Invalid Permission ID
	t.Run("Invalid Permission ID", func(t *testing.T) {
		msg := &types.MsgSetPermissionVPToValidated{
			Authority: validatorAddr,
			Operator:  validatorAddr,
			Id:        9999, // Non-existent ID
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm not found")
		require.Nil(t, resp)
	})

	// 4. Test validation error - Not in PENDING state
	t.Run("Not in PENDING state", func(t *testing.T) {
		// Create a perm that's not in PENDING state
		notPendingPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED, // Not PENDING
		}
		notPendingPermID, err := k.CreatePermission(sdkCtx, notPendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority: validatorAddr,
			Operator:  validatorAddr,
			Id:        notPendingPermID,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "perm must be in PENDING state")
		require.Nil(t, resp)
	})

	// 5. Test validation error - Wrong validator
	t.Run("Wrong validator", func(t *testing.T) {
		// Create a new perm in PENDING state
		pendingPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		pendingPermID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority: otherAddr, // Not the validator
			Operator:  otherAddr,
			Id:        pendingPermID,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authority must be validator perm authority")
		require.Nil(t, resp)
	})

	// 6. Test validation error - HOLDER with digest SRI
	t.Run("HOLDER type with digest SRI", func(t *testing.T) {
		// Create a HOLDER perm in PENDING state
		holderPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_HOLDER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		holderPermID, err := k.CreatePermission(sdkCtx, holderPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      holderPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			VpSummaryDigestSri:      "sha384-someDigest", // Should be empty for HOLDER
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vp_summary_digest_sri must be null for HOLDER type")
		require.Nil(t, resp)
	})

	// 7. Test discount validation - ISSUER_GRANTOR with valid discount
	t.Run("ISSUER_GRANTOR with valid discount", func(t *testing.T) {
		// Create ISSUER_GRANTOR perm in PENDING state
		grantorPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER_GRANTOR,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      grantorPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     5000, // 50% discount
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, grantorPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), updatedPerm.IssuanceFeeDiscount)
	})

	// 8. Test discount validation - ISSUER in GRANTOR mode with discount within validator's limit
	t.Run("ISSUER in GRANTOR mode with valid discount", func(t *testing.T) {
		// First create a validator with a discount
		validatorWithDiscount := types.Permission{
			SchemaId:            1,
			Type:                types.PermissionType_ISSUER_GRANTOR,
			Authority:             validatorAddr,
			Created:             &now,
			CreatedBy:           validatorAddr,
			Extended:            &now,
			ExtendedBy:          validatorAddr,
			Modified:            &now,
			Country:             "US",
			VpState:             types.ValidationState_VALIDATED,
			IssuanceFeeDiscount: 7000,      // 70% discount
			EffectiveFrom:       &pastTime, // Required for ACTIVE state
		}
		validatorWithDiscountID, err := k.CreatePermission(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		// Create ISSUER perm with this validator
		issuerPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorWithDiscountID,
			VpState:         types.ValidationState_PENDING,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		// Can set discount up to validator's discount (7000)
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     5000, // 50% discount (within validator's 70%)
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, issuerPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000), updatedPerm.IssuanceFeeDiscount)
	})

	// 9. Test discount validation - ISSUER in GRANTOR mode exceeding validator's discount
	t.Run("ISSUER in GRANTOR mode exceeding validator discount", func(t *testing.T) {
		// Create ISSUER perm with validator that has 50% discount
		validatorWithDiscount := types.Permission{
			SchemaId:            1,
			Type:                types.PermissionType_ISSUER_GRANTOR,
			Authority:             validatorAddr,
			Created:             &now,
			CreatedBy:           validatorAddr,
			Extended:            &now,
			ExtendedBy:          validatorAddr,
			Modified:            &now,
			Country:             "US",
			VpState:             types.ValidationState_VALIDATED,
			IssuanceFeeDiscount: 5000,      // 50% discount
			EffectiveFrom:       &pastTime, // Required for ACTIVE state
		}
		validatorWithDiscountID, err := k.CreatePermission(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		issuerPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorWithDiscountID,
			VpState:         types.ValidationState_PENDING,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		// Try to set discount exceeding validator's discount
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     6000, // 60% discount (exceeds validator's 50%)
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed validator's discount")
		require.Nil(t, resp)
	})

	// 10. Test discount validation - discount exceeds maximum
	t.Run("Discount exceeds maximum", func(t *testing.T) {
		grantorPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER_GRANTOR,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      grantorPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     10001, // Exceeds maximum of 10000
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed")
		require.Nil(t, resp)
	})

	// 11. Test renewal with discount - must match existing discount
	t.Run("Renewal with discount must match existing", func(t *testing.T) {
		effectiveFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
		renewalPerm := types.Permission{
			SchemaId:            1,
			Type:                types.PermissionType_ISSUER_GRANTOR,
			Authority:             otherAddr, // Use different authority to avoid overlap with test 7
			Created:             &now,
			CreatedBy:           otherAddr,
			Extended:            &now,
			ExtendedBy:          otherAddr,
			Modified:            &now,
			Country:             "US",
			ValidatorPermId:     validatorPermID,
			VpState:             types.ValidationState_PENDING,
			EffectiveFrom:       &effectiveFrom,
			ValidationFees:      10,
			IssuanceFees:        5,
			VerificationFees:    3,
			IssuanceFeeDiscount: 3000, // 30% discount set initially
		}
		renewalPermID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		// Try to change discount during renewal
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      renewalPermID,
			ValidationFees:          10, // Must match
			IssuanceFees:            5,  // Must match
			VerificationFees:        3,  // Must match
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     4000, // Different from existing 3000
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be changed during renewal")
		require.Nil(t, resp)

		// Try with matching discount
		msg.IssuanceFeeDiscount = 3000 // Match existing
		resp, err = ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, renewalPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(3000), updatedPerm.IssuanceFeeDiscount)
	})

	// 12. Test ECOSYSTEM mode - ISSUER with discount
	t.Run("ISSUER in ECOSYSTEM mode with discount", func(t *testing.T) {
		// Create schema with ECOSYSTEM mode
		csKeeper.CreateMockCredentialSchema(2,
			cstypes.CredentialSchemaPermManagementMode_ECOSYSTEM,
			cstypes.CredentialSchemaPermManagementMode_ECOSYSTEM)

		// Create ECOSYSTEM validator
		ecosystemValidator := types.Permission{
			SchemaId:      2,
			Type:          types.PermissionType_ECOSYSTEM,
			Authority:       validatorAddr,
			Created:       &now,
			CreatedBy:     validatorAddr,
			Extended:      &now,
			ExtendedBy:    validatorAddr,
			Modified:      &now,
			Country:       "US",
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime, // Required for ACTIVE state
		}
		ecosystemValidatorID, err := k.CreatePermission(sdkCtx, ecosystemValidator)
		require.NoError(t, err)

		// Create ISSUER perm with ECOSYSTEM validator
		issuerPerm := types.Permission{
			SchemaId:        2,
			Type:            types.PermissionType_ISSUER,
			Authority:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: ecosystemValidatorID,
			VpState:         types.ValidationState_PENDING,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               validatorAddr,
			Operator:                validatorAddr,
			Id:                      issuerPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     8000, // 80% discount (allowed in ECOSYSTEM mode)
			VerificationFeeDiscount: 0,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, issuerPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(8000), updatedPerm.IssuanceFeeDiscount)
	})

	// 13. Test effective_until <= now (first time) should fail
	t.Run("effective_until must be greater than now for first time", func(t *testing.T) {
		euAddr := sdk.AccAddress([]byte("eu_now_creator")).String()
		pendingPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       euAddr,
			Created:         &now,
			CreatedBy:       euAddr,
			Extended:        &now,
			ExtendedBy:      euAddr,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		pastEffUntil := now.Add(-1 * time.Hour) // in the past
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &pastEffUntil,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be greater than current timestamp")
		require.Nil(t, resp)
	})

	// 14. Test effective_until > vp_exp should fail
	t.Run("effective_until must be lower or equal to vp_exp", func(t *testing.T) {
		// Create schema with validity period so vpExp is calculated
		csKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                              3,
			IssuerPermManagementMode:        cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
			VerifierPermManagementMode:      cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
			IssuerValidationValidityPeriod:  30, // 30 days
		})

		// Create validator for schema 3
		vpAddr := sdk.AccAddress([]byte("vp_exp_validator")).String()
		vpValidator := types.Permission{
			SchemaId:      3,
			Type:          types.PermissionType_ISSUER_GRANTOR,
			Authority:     vpAddr,
			Created:       &now,
			CreatedBy:     vpAddr,
			Extended:      &now,
			ExtendedBy:    vpAddr,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vpValidatorID, err := k.CreatePermission(sdkCtx, vpValidator)
		require.NoError(t, err)

		vpTestAddr := sdk.AccAddress([]byte("vp_exp_creator")).String()
		pendingPerm := types.Permission{
			SchemaId:        3,
			Type:            types.PermissionType_ISSUER,
			Authority:       vpTestAddr,
			Created:         &now,
			CreatedBy:       vpTestAddr,
			Extended:        &now,
			ExtendedBy:      vpTestAddr,
			Modified:        &now,
			ValidatorPermId: vpValidatorID,
			VpState:         types.ValidationState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		// vpExp will be now + 30 days. Set effective_until to now + 60 days (beyond vpExp)
		farFuture := now.Add(60 * 24 * time.Hour)
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          vpAddr,
			Operator:           vpAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &farFuture,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be lower or equal to vp_exp")
		require.Nil(t, resp)
	})

	// 15. Test effective_until nil resolves to vpExp
	t.Run("effective_until nil resolves to vp_exp", func(t *testing.T) {
		// Schema 3 already has 30-day validity period from test 14
		vpAddr := sdk.AccAddress([]byte("vp_exp_validator")).String()
		// Find the validator perm ID for schema 3
		vpNilAddr := sdk.AccAddress([]byte("vp_nil_creator")).String()

		// Create validator for schema 3 (separate to avoid overlap)
		vpValidator2 := types.Permission{
			SchemaId:      3,
			Type:          types.PermissionType_ISSUER_GRANTOR,
			Authority:     vpAddr,
			Created:       &now,
			CreatedBy:     vpAddr,
			Extended:      &now,
			ExtendedBy:    vpAddr,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vpValidator2ID, err := k.CreatePermission(sdkCtx, vpValidator2)
		require.NoError(t, err)

		pendingPerm := types.Permission{
			SchemaId:        3,
			Type:            types.PermissionType_ISSUER,
			Authority:       vpNilAddr,
			Created:         &now,
			CreatedBy:       vpNilAddr,
			Extended:        &now,
			ExtendedBy:      vpNilAddr,
			Modified:        &now,
			ValidatorPermId: vpValidator2ID,
			VpState:         types.ValidationState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          vpAddr,
			Operator:           vpAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     nil, // nil should resolve to vpExp
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, permID)
		require.NoError(t, err)
		// effective_until should equal vpExp (now + 30 days)
		expectedVpExp := now.AddDate(0, 0, 30)
		require.NotNil(t, updatedPerm.VpExp)
		require.Equal(t, expectedVpExp.Unix(), updatedPerm.VpExp.Unix())
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, expectedVpExp.Unix(), updatedPerm.EffectiveUntil.Unix())
	})

	// 16. Test renewal effective_until must be greater than current effective_until
	t.Run("Renewal effective_until must be greater than current", func(t *testing.T) {
		renewAddr := sdk.AccAddress([]byte("renew_eu_creator")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(30 * 24 * time.Hour) // 30 days in future
		renewalPerm := types.Permission{
			SchemaId:         1,
			Type:             types.PermissionType_ISSUER,
			Authority:        renewAddr,
			Created:          &now,
			CreatedBy:        renewAddr,
			Extended:         &now,
			ExtendedBy:       renewAddr,
			Modified:         &now,
			Country:          "US",
			ValidatorPermId:  validatorPermID,
			VpState:          types.ValidationState_PENDING,
			EffectiveFrom:    &effectiveFrom,
			EffectiveUntil:   &currentEffUntil,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		// Try with effective_until <= current effective_until
		smallerEffUntil := now.Add(10 * 24 * time.Hour)
		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &smallerEffUntil,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "effective_until must be greater than current effective_until")
		require.Nil(t, resp)
	})

	// 17. Test renewal validation_fees mismatch
	t.Run("Renewal validation_fees must match", func(t *testing.T) {
		rvfAddr := sdk.AccAddress([]byte("ren_valfees_addr")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Permission{
			SchemaId:         1,
			Type:             types.PermissionType_ISSUER,
			Authority:        rvfAddr,
			Created:          &now,
			CreatedBy:        rvfAddr,
			Extended:         &now,
			ExtendedBy:       rvfAddr,
			Modified:         &now,
			Country:          "US",
			ValidatorPermId:  validatorPermID,
			VpState:          types.ValidationState_PENDING,
			EffectiveFrom:    &effectiveFrom,
			EffectiveUntil:   &currentEffUntil,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     20, // Different from existing 10
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation_fees cannot be changed during renewal")
		require.Nil(t, resp)
	})

	// 18. Test renewal issuance_fees mismatch
	t.Run("Renewal issuance_fees must match", func(t *testing.T) {
		rifAddr := sdk.AccAddress([]byte("ren_issfees_addr")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Permission{
			SchemaId:         1,
			Type:             types.PermissionType_ISSUER,
			Authority:        rifAddr,
			Created:          &now,
			CreatedBy:        rifAddr,
			Extended:         &now,
			ExtendedBy:       rifAddr,
			Modified:         &now,
			Country:          "US",
			ValidatorPermId:  validatorPermID,
			VpState:          types.ValidationState_PENDING,
			EffectiveFrom:    &effectiveFrom,
			EffectiveUntil:   &currentEffUntil,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       99, // Different from existing 5
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "issuance_fees cannot be changed during renewal")
		require.Nil(t, resp)
	})

	// 19. Test renewal verification_fees mismatch
	t.Run("Renewal verification_fees must match", func(t *testing.T) {
		rvAddr := sdk.AccAddress([]byte("ren_verfees_addr")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Permission{
			SchemaId:         1,
			Type:             types.PermissionType_ISSUER,
			Authority:        rvAddr,
			Created:          &now,
			CreatedBy:        rvAddr,
			Extended:         &now,
			ExtendedBy:       rvAddr,
			Modified:         &now,
			Country:          "US",
			ValidatorPermId:  validatorPermID,
			VpState:          types.ValidationState_PENDING,
			EffectiveFrom:    &effectiveFrom,
			EffectiveUntil:   &currentEffUntil,
			ValidationFees:   10,
			IssuanceFees:     5,
			VerificationFees: 3,
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   99, // Different from existing 3
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "verification_fees cannot be changed during renewal")
		require.Nil(t, resp)
	})

	// 20. Test overlap - existing perm never expires (nil effective_until)
	t.Run("Overlap with never-expiring permission", func(t *testing.T) {
		overlapAddr := sdk.AccAddress([]byte("overlap_never_addr")).String()
		// Create an existing validated perm with nil effective_until (never expires)
		existingPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       overlapAddr,
			Created:         &now,
			CreatedBy:       overlapAddr,
			Extended:        &now,
			ExtendedBy:      overlapAddr,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED,
			EffectiveFrom:   &pastTime,
			EffectiveUntil:  nil, // Never expires
		}
		_, err := k.CreatePermission(sdkCtx, existingPerm)
		require.NoError(t, err)

		// Try to create a new validated perm with same (schema, type, validator, authority)
		newPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       overlapAddr,
			Created:         &now,
			CreatedBy:       overlapAddr,
			Extended:        &now,
			ExtendedBy:      overlapAddr,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 newPermID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
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
		existingPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       overlapAddr2,
			Created:         &now,
			CreatedBy:       overlapAddr2,
			Extended:        &now,
			ExtendedBy:      overlapAddr2,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED,
			EffectiveFrom:   &pastTime,
			EffectiveUntil:  &existingEffUntil,
		}
		_, err := k.CreatePermission(sdkCtx, existingPerm)
		require.NoError(t, err)

		// Try to validate a new perm — effective_from will be set to now, which is before existing's effective_until
		newPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       overlapAddr2,
			Created:         &now,
			CreatedBy:       overlapAddr2,
			Extended:        &now,
			ExtendedBy:      overlapAddr2,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 newPermID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap check failed")
		require.Nil(t, resp)
	})

	// 22. Test validator perm not active (revoked)
	t.Run("Validator perm is revoked", func(t *testing.T) {
		revokedTime := now.Add(-1 * time.Hour)
		revokedValidator := types.Permission{
			SchemaId:      1,
			Type:          types.PermissionType_ISSUER_GRANTOR,
			Authority:     validatorAddr,
			Created:       &now,
			CreatedBy:     validatorAddr,
			Extended:      &now,
			ExtendedBy:    validatorAddr,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
			Revoked:       &revokedTime,
		}
		revokedValidatorID, err := k.CreatePermission(sdkCtx, revokedValidator)
		require.NoError(t, err)

		rvAddr := sdk.AccAddress([]byte("revoked_val_addr")).String()
		pendingPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       rvAddr,
			Created:         &now,
			CreatedBy:       rvAddr,
			Extended:        &now,
			ExtendedBy:      rvAddr,
			Modified:        &now,
			ValidatorPermId: revokedValidatorID,
			VpState:         types.ValidationState_PENDING,
		}
		permID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator perm is not valid")
		require.Nil(t, resp)
	})

	// 23. Test with VpCurrentFees and VpCurrentDeposit > 0 (execution: fee transfer + trust deposit)
	t.Run("Execution with fees and trust deposit", func(t *testing.T) {
		feeAddr := sdk.AccAddress([]byte("fee_exec_creator")).String()
		newPerm := types.Permission{
			SchemaId:         1,
			Type:             types.PermissionType_ISSUER,
			Authority:        feeAddr,
			Created:          &now,
			CreatedBy:        feeAddr,
			Extended:         &now,
			ExtendedBy:       feeAddr,
			Modified:         &now,
			Country:          "US",
			ValidatorPermId:  validatorPermID,
			VpState:          types.ValidationState_PENDING,
			VpCurrentFees:    100, // Has fees to transfer
			VpCurrentDeposit: 50,  // Has deposit to transfer
		}
		permID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 permID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, permID)
		require.NoError(t, err)
		require.Equal(t, uint64(0), updatedPerm.VpCurrentFees)    // Reset to 0
		require.Equal(t, uint64(0), updatedPerm.VpCurrentDeposit) // Reset to 0
		require.Equal(t, uint64(50), updatedPerm.VpValidatorDeposit) // Accumulated
	})

	// 24. Test VERIFIER_GRANTOR with verification_fee_discount
	t.Run("VERIFIER_GRANTOR with verification_fee_discount", func(t *testing.T) {
		// Create schema with GRANTOR_VALIDATION for verifier mode
		csKeeper.CreateMockCredentialSchemaFull(cstypes.CredentialSchema{
			Id:                              4,
			IssuerPermManagementMode:        cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
			VerifierPermManagementMode:      cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		})

		vgAddr := sdk.AccAddress([]byte("ver_grantor_vali")).String()
		vgValidator := types.Permission{
			SchemaId:      4,
			Type:          types.PermissionType_VERIFIER_GRANTOR,
			Authority:     vgAddr,
			Created:       &now,
			CreatedBy:     vgAddr,
			Extended:      &now,
			ExtendedBy:    vgAddr,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgValidatorID, err := k.CreatePermission(sdkCtx, vgValidator)
		require.NoError(t, err)

		// Create VERIFIER_GRANTOR perm (can set its own verification_fee_discount)
		vgPerm := types.Permission{
			SchemaId:        4,
			Type:            types.PermissionType_VERIFIER_GRANTOR,
			Authority:       sdk.AccAddress([]byte("vg_perm_creator")).String(),
			Created:         &now,
			CreatedBy:       sdk.AccAddress([]byte("vg_perm_creator")).String(),
			Extended:        &now,
			ExtendedBy:      sdk.AccAddress([]byte("vg_perm_creator")).String(),
			Modified:        &now,
			ValidatorPermId: vgValidatorID,
			VpState:         types.ValidationState_PENDING,
		}
		vgPermID, err := k.CreatePermission(sdkCtx, vgPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               vgAddr,
			Operator:                vgAddr,
			Id:                      vgPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 6000, // 60% discount
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		updatedPerm, err := k.GetPermissionByID(sdkCtx, vgPermID)
		require.NoError(t, err)
		require.Equal(t, uint64(6000), updatedPerm.VerificationFeeDiscount)
	})

	// 25. Test VERIFIER in GRANTOR mode exceeding validator's verification_fee_discount
	t.Run("VERIFIER exceeding validator verification_fee_discount", func(t *testing.T) {
		// Create validator with verification_fee_discount
		vgAddr2 := sdk.AccAddress([]byte("vg_disc_validato")).String()
		vgValidator2 := types.Permission{
			SchemaId:                 4,
			Type:                     types.PermissionType_VERIFIER_GRANTOR,
			Authority:                vgAddr2,
			Created:                  &now,
			CreatedBy:                vgAddr2,
			Extended:                 &now,
			ExtendedBy:               vgAddr2,
			Modified:                 &now,
			VpState:                  types.ValidationState_VALIDATED,
			VerificationFeeDiscount:  5000, // 50% discount
			EffectiveFrom:            &pastTime,
		}
		vgValidator2ID, err := k.CreatePermission(sdkCtx, vgValidator2)
		require.NoError(t, err)

		verPerm := types.Permission{
			SchemaId:        4,
			Type:            types.PermissionType_VERIFIER,
			Authority:       sdk.AccAddress([]byte("ver_exceed_addr")).String(),
			Created:         &now,
			CreatedBy:       sdk.AccAddress([]byte("ver_exceed_addr")).String(),
			Extended:        &now,
			ExtendedBy:      sdk.AccAddress([]byte("ver_exceed_addr")).String(),
			Modified:        &now,
			ValidatorPermId: vgValidator2ID,
			VpState:         types.ValidationState_PENDING,
		}
		verPermID, err := k.CreatePermission(sdkCtx, verPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               vgAddr2,
			Operator:                vgAddr2,
			Id:                      verPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 7000, // 70% exceeds validator's 50%
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed validator's discount")
		require.Nil(t, resp)
	})

	// 26. Test overlap check skips revoked permissions
	t.Run("Overlap check skips revoked permissions", func(t *testing.T) {
		skipAddr := sdk.AccAddress([]byte("overlap_skip_addr")).String()
		revokedTime := now.Add(-1 * time.Hour)
		// Create a revoked perm (should be skipped in overlap check)
		revokedPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       skipAddr,
			Created:         &now,
			CreatedBy:       skipAddr,
			Extended:        &now,
			ExtendedBy:      skipAddr,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATED,
			EffectiveFrom:   &pastTime,
			EffectiveUntil:  nil,     // Never expires, but it's revoked
			Revoked:         &revokedTime,
		}
		_, err := k.CreatePermission(sdkCtx, revokedPerm)
		require.NoError(t, err)

		// This new perm should NOT conflict with the revoked one
		newPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_ISSUER,
			Authority:       skipAddr,
			Created:         &now,
			CreatedBy:       skipAddr,
			Extended:        &now,
			ExtendedBy:      skipAddr,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:          validatorAddr,
			Operator:           validatorAddr,
			Id:                 newPermID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 27. Test verification_fee_discount exceeds maximum
	t.Run("Verification_fee_discount exceeds maximum", func(t *testing.T) {
		vfdAddr := sdk.AccAddress([]byte("vfd_max_creator")).String()
		grantorPerm := types.Permission{
			SchemaId:        4,
			Type:            types.PermissionType_VERIFIER_GRANTOR,
			Authority:       vfdAddr,
			Created:         &now,
			CreatedBy:       vfdAddr,
			Extended:        &now,
			ExtendedBy:      vfdAddr,
			Modified:        &now,
			ValidatorPermId: func() uint64 {
				// Reuse a schema 4 validator
				vAddr := sdk.AccAddress([]byte("vfd_max_validato")).String()
				v := types.Permission{
					SchemaId:      4,
					Type:          types.PermissionType_VERIFIER_GRANTOR,
					Authority:     vAddr,
					Created:       &now,
					CreatedBy:     vAddr,
					Extended:      &now,
					ExtendedBy:    vAddr,
					Modified:      &now,
					VpState:       types.ValidationState_VALIDATED,
					EffectiveFrom: &pastTime,
				}
				id, _ := k.CreatePermission(sdkCtx, v)
				return id
			}(),
			VpState: types.ValidationState_PENDING,
		}
		grantorPermID, err := k.CreatePermission(sdkCtx, grantorPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               sdk.AccAddress([]byte("vfd_max_validato")).String(),
			Operator:                sdk.AccAddress([]byte("vfd_max_validato")).String(),
			Id:                      grantorPermID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 10001, // Exceeds maximum
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot exceed")
		require.Nil(t, resp)
	})

	// 28. Test renewal discount for verification_fee_discount must match
	t.Run("Renewal verification_fee_discount must match existing", func(t *testing.T) {
		rvdAddr := sdk.AccAddress([]byte("ren_vfd_creator")).String()
		effectiveFrom := now.Add(-90 * 24 * time.Hour)
		currentEffUntil := now.Add(-1 * time.Hour)
		renewalPerm := types.Permission{
			SchemaId:                4,
			Type:                    types.PermissionType_VERIFIER_GRANTOR,
			Authority:               rvdAddr,
			Created:                 &now,
			CreatedBy:               rvdAddr,
			Extended:                &now,
			ExtendedBy:              rvdAddr,
			Modified:                &now,
			ValidatorPermId: func() uint64 {
				vAddr := sdk.AccAddress([]byte("rvd_validator_ad")).String()
				v := types.Permission{
					SchemaId:      4,
					Type:          types.PermissionType_VERIFIER_GRANTOR,
					Authority:     vAddr,
					Created:       &now,
					CreatedBy:     vAddr,
					Extended:      &now,
					ExtendedBy:    vAddr,
					Modified:      &now,
					VpState:       types.ValidationState_VALIDATED,
					EffectiveFrom: &pastTime,
				}
				id, _ := k.CreatePermission(sdkCtx, v)
				return id
			}(),
			VpState:                 types.ValidationState_PENDING,
			EffectiveFrom:           &effectiveFrom,
			EffectiveUntil:          &currentEffUntil,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			VerificationFeeDiscount: 4000, // Existing 40%
		}
		permID, err := k.CreatePermission(sdkCtx, renewalPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Authority:               sdk.AccAddress([]byte("rvd_validator_ad")).String(),
			Operator:                sdk.AccAddress([]byte("rvd_validator_ad")).String(),
			Id:                      permID,
			ValidationFees:          10,
			IssuanceFees:            5,
			VerificationFees:        3,
			EffectiveUntil:          &futureTime,
			VpSummaryDigestSri:      "sha384-validDigest",
			IssuanceFeeDiscount:     0,
			VerificationFeeDiscount: 6000, // Different from existing 4000
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "verification_fee_discount cannot be changed during renewal")
		require.Nil(t, resp)
	})
}

// Test AUTHZ-CHECK failure for SetPermissionVPToValidated
func TestSetPermissionVPToValidated_AuthzCheckFailure(t *testing.T) {
	k, ms, csKeeper, _, ctx, delKeeper := setupMsgServerWithDelegation(t)
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
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:     validatorAddr,
		Created:       &now,
		CreatedBy:     validatorAddr,
		Extended:      &now,
		ExtendedBy:    validatorAddr,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create perm to validate
	pendingPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:       creatorAddr,
		Created:         &now,
		CreatedBy:       creatorAddr,
		Extended:        &now,
		ExtendedBy:      creatorAddr,
		Modified:        &now,
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_PENDING,
	}
	permID, err := k.CreatePermission(sdkCtx, pendingPerm)
	require.NoError(t, err)

	// Set delegation keeper to return error
	delKeeper.ErrToReturn = fmt.Errorf("operator not authorized")

	msg := &types.MsgSetPermissionVPToValidated{
		Authority:          validatorAddr,
		Operator:           operatorAddr,
		Id:                 permID,
		ValidationFees:     10,
		IssuanceFees:       5,
		VerificationFees:   3,
		EffectiveUntil:     &futureTime,
		VpSummaryDigestSri: "sha384-validDigest",
	}

	resp, err := ms.SetPermissionVPToValidated(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization check failed")
	require.Contains(t, err.Error(), "operator not authorized")
	require.Nil(t, resp)

	// Reset to allow and verify it works
	delKeeper.ErrToReturn = nil

	// Need to use validatorAddr as authority (not creatorAddr) since validator perm authority check
	resp, err = ms.SetPermissionVPToValidated(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestMsgServerCreateRootPermission(t *testing.T) {
	k, ms, mockCsKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry and store its ID
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema with specific perm management modes and trust registry ID
	mockCsKeeper.UpdateMockCredentialSchema(1,
		trID, // Set the trust registry ID
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()
	futureTime := now.Add(24 * time.Hour)

	testCases := []struct {
		name    string
		msg     *types.MsgCreateRootPermission
		isValid bool
	}{
		{
			name: "Valid Create Root Permission",
			msg: &types.MsgCreateRootPermission{
				Creator:          creator,
				SchemaId:         1,
				Did:              validDid,
				ValidationFees:   100,
				IssuanceFees:     50,
				VerificationFees: 25,
				Country:          "US",
				EffectiveFrom:    &now,
				EffectiveUntil:   &futureTime,
			},
			isValid: true,
		},
		{
			name: "Non-existent Schema ID",
			msg: &types.MsgCreateRootPermission{
				Creator:          creator,
				SchemaId:         999,
				Did:              validDid,
				ValidationFees:   100,
				IssuanceFees:     50,
				VerificationFees: 25,
			},
			isValid: false,
		},
		{
			name: "Wrong Creator (Not Trust Registry Controller)",
			msg: &types.MsgCreateRootPermission{
				Creator:          sdk.AccAddress([]byte("wrong_creator")).String(),
				SchemaId:         1,
				Did:              validDid,
				ValidationFees:   100,
				IssuanceFees:     50,
				VerificationFees: 25,
			},
			isValid: false,
		},
	}

	var expectedID uint64 = 1 // Track expected auto-generated ID

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CreateRootPermission(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify ID was auto-generated correctly
				require.Equal(t, expectedID, resp.Id)

				// Get the created perm
				perm, err := k.GetPermissionByID(sdkCtx, resp.Id)
				require.NoError(t, err)

				// Verify all fields are set correctly
				require.Equal(t, tc.msg.SchemaId, perm.SchemaId)
				require.Equal(t, tc.msg.Did, perm.Did)
				require.Equal(t, tc.msg.Creator, perm.Authority)
				require.Equal(t, types.PermissionType_ECOSYSTEM, perm.Type)
				require.Equal(t, tc.msg.ValidationFees, perm.ValidationFees)
				require.Equal(t, tc.msg.IssuanceFees, perm.IssuanceFees)
				require.Equal(t, tc.msg.VerificationFees, perm.VerificationFees)
				require.Equal(t, tc.msg.Country, perm.Country)

				// Verify time fields if set
				if tc.msg.EffectiveFrom != nil {
					require.Equal(t, tc.msg.EffectiveFrom.Unix(), perm.EffectiveFrom.Unix())
				}
				if tc.msg.EffectiveUntil != nil {
					require.Equal(t, tc.msg.EffectiveUntil.Unix(), perm.EffectiveUntil.Unix())
				}

				// Verify auto-populated fields
				require.NotNil(t, perm.Created)
				require.NotNil(t, perm.Modified)
				require.Equal(t, tc.msg.Creator, perm.CreatedBy)

				expectedID++ // Increment expected ID for next valid creation
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestCancelPermissionVPLastRequest(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Use the block time for permissions
	now := sdkCtx.BlockTime()

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_ISSUER_GRANTOR,
		Authority:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm in PENDING state that has never been validated (vp_exp is nil)
	// This should transition to TERMINATED when cancelled
	neverValidatedPerm := types.Permission{
		SchemaId:         1,
		Type:             types.PermissionType_ISSUER,
		Authority:          creator,
		Created:          &now,
		CreatedBy:        creator,
		Extended:         &now,
		ExtendedBy:       creator,
		Modified:         &now,
		Country:          "US",
		ValidatorPermId:  validatorPermID,
		VpState:          types.ValidationState_PENDING,
		VpCurrentFees:    100,
		VpCurrentDeposit: 50,
		// VpExp is nil, indicating it has never been validated
	}
	neverValidatedPermID, err := k.CreatePermission(sdkCtx, neverValidatedPerm)
	require.NoError(t, err)

	// Create a perm in PENDING state with a previous validation (has VpExp)
	// This should transition to VALIDATED when cancelled
	futureTime := now.Add(24 * time.Hour)
	previouslyValidatedPerm := types.Permission{
		SchemaId:         1,
		Type:             types.PermissionType_ISSUER,
		Authority:          creator,
		Created:          &now,
		CreatedBy:        creator,
		Extended:         &now,
		ExtendedBy:       creator,
		Modified:         &now,
		Country:          "US",
		ValidatorPermId:  validatorPermID,
		VpState:          types.ValidationState_PENDING,
		VpExp:            &futureTime, // Has a previous validation
		VpCurrentFees:    100,
		VpCurrentDeposit: 50,
	}
	previouslyValidatedPermID, err := k.CreatePermission(sdkCtx, previouslyValidatedPerm)
	require.NoError(t, err)

	// Create a perm not in PENDING state for testing validation error
	notPendingPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED, // Not in PENDING state
	}
	notPendingPermID, err := k.CreatePermission(sdkCtx, notPendingPerm)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgCancelPermissionVPLastRequest
		expectErr  bool
		errMessage string
		checkState bool
		expState   types.ValidationState
	}{
		{
			name: "Valid cancellation - never validated before",
			msg: &types.MsgCancelPermissionVPLastRequest{
				Creator: creator,
				Id:      neverValidatedPermID,
			},
			expectErr:  false,
			checkState: true,
			expState:   types.ValidationState_TERMINATED,
		},
		{
			name: "Valid cancellation - previously validated",
			msg: &types.MsgCancelPermissionVPLastRequest{
				Creator: creator,
				Id:      previouslyValidatedPermID,
			},
			expectErr:  false,
			checkState: true,
			expState:   types.ValidationState_VALIDATED,
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgCancelPermissionVPLastRequest{
				Creator: creator,
				Id:      9999,
			},
			expectErr:  true,
			errMessage: "perm not found",
		},
		{
			name: "Invalid - wrong creator",
			msg: &types.MsgCancelPermissionVPLastRequest{
				Creator: validatorAddr, // Not the perm grantee
				Id:      neverValidatedPermID,
			},
			expectErr:  true,
			errMessage: "creator is not the perm authority",
		},
		{
			name: "Invalid - not in PENDING state",
			msg: &types.MsgCancelPermissionVPLastRequest{
				Creator: creator,
				Id:      notPendingPermID, // Not in PENDING state
			},
			expectErr:  true,
			errMessage: "perm must be in PENDING state",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CancelPermissionVPLastRequest(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				if tc.checkState {
					// Verify perm state was updated correctly
					perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
					require.NoError(t, err)
					require.Equal(t, tc.expState, perm.VpState)

					// Check that fees and deposits were properly returned
					require.Equal(t, uint64(0), perm.VpCurrentFees)
					require.Equal(t, uint64(0), perm.VpCurrentDeposit)
				}
			}
		})
	}
}

// TestExtendPermission tests the ExtendPermission message server function
func TestExtendPermission(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	trustRegistryAddr := sdk.AccAddress([]byte("trust_registry")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	currentEffectiveUntil := now.Add(30 * 24 * time.Hour) // 30 days in the future
	futureVpExp := now.Add(365 * 24 * time.Hour)          // 1 year in the future
	pastTime := now.Add(-1 * time.Hour)                   // Set effective_from to past to make it ACTIVE

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       validatorAddr,
		Created:       &now,
		CreatedBy:     validatorAddr,
		Extended:      &now,
		ExtendedBy:    validatorAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm to extend
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  &currentEffectiveUntil,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		VpExp:           &futureVpExp,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)

	require.NoError(t, err)

	// Create a trust registry perm to test direct extension
	trustRegistryPerm := types.Permission{
		SchemaId:       1,
		Type:           types.PermissionType_ECOSYSTEM,
		Authority:        trustRegistryAddr,
		Created:        &now,
		CreatedBy:      trustRegistryAddr,
		Extended:       &now,
		ExtendedBy:     trustRegistryAddr,
		Modified:       &now,
		EffectiveUntil: &currentEffectiveUntil,
		Country:        "US",
		VpState:        types.ValidationState_VALIDATED,
		EffectiveFrom:  &pastTime, // Required for ACTIVE state
	}
	trustRegistryPermID, err := k.CreatePermission(sdkCtx, trustRegistryPerm)
	require.NoError(t, err)

	// Create a separate perm for the "wrong creator" test
	// Use same validator but has a different effective_until date
	wrongCreatorTestPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  &currentEffectiveUntil, // Same as the regular test
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		VpExp:           &futureVpExp,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	wrongCreatorTestPermID, err := k.CreatePermission(sdkCtx, wrongCreatorTestPerm)
	require.NoError(t, err)

	// Create a perm with NULL effective_until for testing NULL case
	nullEffectiveUntilPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  nil, // NULL effective_until
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		VpExp:           &futureVpExp,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	nullEffectiveUntilPermID, err := k.CreatePermission(sdkCtx, nullEffectiveUntilPerm)
	require.NoError(t, err)

	// Create an ecosystem perm with NULL effective_until
	nullEffectiveUntilEcosystemPerm := types.Permission{
		SchemaId:       1,
		Type:           types.PermissionType_ECOSYSTEM,
		Authority:        trustRegistryAddr,
		Created:        &now,
		CreatedBy:      trustRegistryAddr,
		Extended:       &now,
		ExtendedBy:     trustRegistryAddr,
		Modified:       &now,
		EffectiveUntil: nil, // NULL effective_until
		Country:        "US",
		VpState:        types.ValidationState_VALIDATED,
		EffectiveFrom:  &pastTime, // Required for ACTIVE state
	}
	nullEffectiveUntilEcosystemPermID, err := k.CreatePermission(sdkCtx, nullEffectiveUntilEcosystemPerm)
	require.NoError(t, err)

	// Create additional perms with NULL effective_until for invalid test cases
	// (each test case needs its own perm since extending modifies the state)
	nullEffectiveUntilPermForPastTest := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  nil, // NULL effective_until
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		VpExp:           &futureVpExp,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	nullEffectiveUntilPermForPastTestID, err := k.CreatePermission(sdkCtx, nullEffectiveUntilPermForPastTest)
	require.NoError(t, err)

	nullEffectiveUntilPermForEqualTest := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  nil, // NULL effective_until
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		VpExp:           &futureVpExp,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	nullEffectiveUntilPermForEqualTestID, err := k.CreatePermission(sdkCtx, nullEffectiveUntilPermForEqualTest)
	require.NoError(t, err)

	newEffectiveUntil := now.Add(60 * 24 * time.Hour)     // 60 days in the future
	pastEffectiveUntil := now.Add(-1 * 24 * time.Hour)    // 1 day in the past
	tooFarEffectiveUntil := now.Add(500 * 24 * time.Hour) // Past VP expiration
	equalToNowEffectiveUntil := now                       // Equal to now (should fail)

	testCases := []struct {
		name       string
		msg        *types.MsgExtendPermission
		expectErr  bool
		errMessage string
	}{
		{
			name: "Valid extension by validator",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Valid extension by trust registry controller",
			msg: &types.MsgExtendPermission{
				Creator:        trustRegistryAddr,
				Id:             trustRegistryPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             9999,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "permission not found",
		},
		{
			name: "Invalid - effective_until not after current effective_until",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &currentEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current effective_until",
		},
		{
			name: "Invalid - effective_until in the past",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &pastEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current effective_until",
		},
		{
			name: "Invalid - effective_until beyond validation expiration",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &tooFarEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until cannot be after validation expiration",
		},
		{
			name: "Invalid - wrong creator",
			msg: &types.MsgExtendPermission{
				Creator:        creator,
				Id:             wrongCreatorTestPermID, // Using separate test perm
				EffectiveUntil: &newEffectiveUntil,     // Valid future time
			},
			expectErr:  true,
			errMessage: "creator is not the validator",
		},
		{
			name: "Valid - extend permission with NULL effective_until (validator-managed)",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             nullEffectiveUntilPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Valid - extend permission with NULL effective_until (ecosystem)",
			msg: &types.MsgExtendPermission{
				Creator:        trustRegistryAddr,
				Id:             nullEffectiveUntilEcosystemPermID,
				EffectiveUntil: &newEffectiveUntil,
			},
			expectErr: false,
		},
		{
			name: "Invalid - extend permission with NULL effective_until but new effective_until not greater than now (past)",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             nullEffectiveUntilPermForPastTestID,
				EffectiveUntil: &pastEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current timestamp",
		},
		{
			name: "Invalid - extend permission with NULL effective_until but new effective_until equals now",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             nullEffectiveUntilPermForEqualTestID,
				EffectiveUntil: &equalToNowEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be greater than current timestamp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.ExtendPermission(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm was extended
				perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.EffectiveUntil.Unix(), perm.EffectiveUntil.Unix())
				require.Equal(t, tc.msg.Creator, perm.ExtendedBy)
				require.NotNil(t, perm.Extended)
			}
		})
	}
}

// TestRevokePermission tests the RevokePermission message server function
func TestRevokePermission(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       validatorAddr,
		Created:       &now,
		CreatedBy:     validatorAddr,
		Extended:      &now,
		ExtendedBy:    validatorAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // Required for ACTIVE state
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm to revoke
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}

	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgRevokePermission
		expectErr  bool
		errMessage string
	}{
		{
			name: "Valid revocation by validator",
			msg: &types.MsgRevokePermission{
				Creator: validatorAddr,
				Id:      applicantPermID,
			},
			expectErr: false,
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgRevokePermission{
				Creator: validatorAddr,
				Id:      9999,
			},
			expectErr:  true,
			errMessage: "permission not found",
		},
		//{
		//	name: "Invalid - validator not found",
		//	msg: &types.MsgRevokePermission{
		//		Creator: validatorAddr,
		//		Id:      validatorPermID, // Validator perm has no validator
		//	},
		//	expectErr:  true,
		//	errMessage: "validator permission not found",
		//},
		//{
		//	name: "Invalid - wrong creator",
		//	msg: &types.MsgRevokePermission{
		//		Creator: creator,
		//		Id:      applicantPermID,
		//	},
		//	expectErr:  true,
		//	errMessage: "creator is not the validator",
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RevokePermission(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm was revoked
				perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, perm.Revoked)
				require.Equal(t, tc.msg.Creator, perm.RevokedBy)
			}
		})
	}
}

func TestCreateOrUpdatePermissionSession(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	sessionUUID := uuid.New().String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Note: We're not calling the setter methods since they don't exist
	// Instead, we'll rely on whatever default values the mock implementations return
	// If you want to test with specific values, you'll need to implement Option 1

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past to make it ACTIVE

	// Create trust registry / validator perm
	trustPerm := types.Permission{
		SchemaId:         1,
		Type:             types.PermissionType_ECOSYSTEM,
		Authority:          creator,
		Created:          &now,
		CreatedBy:        creator,
		Extended:         &now,
		ExtendedBy:       creator,
		Modified:         &now,
		Country:          "US",
		VpState:          types.ValidationState_VALIDATED,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
		EffectiveFrom:    &pastTime, // Required for ACTIVE state
	}
	trustPermID, err := k.CreatePermission(sdkCtx, trustPerm)
	require.NoError(t, err)

	// Create issuer perm
	issuerPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: trustPermID,
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Create verifier perm
	//verifierPerm := types.Permission{
	//	SchemaId:        1,
	//	Type:            types.PermissionType_PERMISSION_TYPE_VERIFIER,
	//	Authority:         creator,
	//	Created:         &now,
	//	CreatedBy:       creator,
	//	Extended:        &now,
	//	ExtendedBy:      creator,
	//	Modified:        &now,
	//	Country:         "US",
	//	ValidatorPermId: trustPermID,
	//	VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	//}
	//verifierPermID, err := k.CreatePermission(sdkCtx, verifierPerm)
	//require.NoError(t, err)

	// Create agent perm (HOLDER type)
	agentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID, // Issued by the issuer
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	// Create wallet agent perm (HOLDER type)
	walletAgentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID, // Issued by the issuer
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime, // Required for ACTIVE state
	}

	walletAgentPermID, err := k.CreatePermission(sdkCtx, walletAgentPerm)
	require.NoError(t, err)

	// Create revoked perm
	//revokedPerm := types.Permission{
	//	SchemaId:        1,
	//	Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
	//	Authority:         creator,
	//	Created:         &now,
	//	CreatedBy:       creator,
	//	Extended:        &now,
	//	ExtendedBy:      creator,
	//	Modified:        &now,
	//	Country:         "US",
	//	ValidatorPermId: trustPermID,
	//	VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	//	Revoked:         &now,
	//	RevokedBy:       creator,
	//}
	//revokedPermID, err := k.CreatePermission(sdkCtx, revokedPerm)
	//require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgCreateOrUpdatePermissionSession
		expectErr  bool
		errMessage string
	}{
		{
			name: "Valid create session with issuer",
			msg: &types.MsgCreateOrUpdatePermissionSession{
				Creator:           creator,
				Id:                sessionUUID,
				IssuerPermId:      issuerPermID,
				VerifierPermId:    0,
				AgentPermId:       agentPermID,
				WalletAgentPermId: walletAgentPermID,
			},
			expectErr: false,
		},
		//{
		//	name: "Valid create session with verifier",
		//	msg: &types.MsgCreateOrUpdatePermissionSession{
		//		Creator:           creator,
		//		Id:                uuid.New().String(),
		//		IssuerPermId:      0,
		//		VerifierPermId:    verifierPermID,
		//		AgentPermId:       agentPermID,
		//		WalletAgentPermId: walletAgentPermID,
		//	},
		//	expectErr: false,
		//},
		//{
		//	name: "Valid create session with both issuer and verifier",
		//	msg: &types.MsgCreateOrUpdatePermissionSession{
		//		Creator:           creator,
		//		Id:                uuid.New().String(),
		//		IssuerPermId:      issuerPermID,
		//		VerifierPermId:    verifierPermID,
		//		AgentPermId:       agentPermID,
		//		WalletAgentPermId: walletAgentPermID,
		//	},
		//	expectErr: false,
		//},
		//{
		//	name: "Valid update existing session",
		//	msg: &types.MsgCreateOrUpdatePermissionSession{
		//		Creator:           creator,
		//		Id:                sessionUUID,
		//		IssuerPermId:      0,
		//		VerifierPermId:    verifierPermID,
		//		AgentPermId:       agentPermID,
		//		WalletAgentPermId: walletAgentPermID,
		//	},
		//	expectErr: false,
		//},
		{
			name: "Invalid - issuer perm not found",
			msg: &types.MsgCreateOrUpdatePermissionSession{
				Creator:           creator,
				Id:                uuid.New().String(),
				IssuerPermId:      9999,
				VerifierPermId:    0,
				AgentPermId:       agentPermID,
				WalletAgentPermId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "issuer permission not found",
		},
		{
			name: "Invalid - invalid issuer type",
			msg: &types.MsgCreateOrUpdatePermissionSession{
				Creator:           creator,
				Id:                uuid.New().String(),
				IssuerPermId:      trustPermID, // Not ISSUER type
				VerifierPermId:    0,
				AgentPermId:       agentPermID,
				WalletAgentPermId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "issuer permission must be ISSUER type",
		},
		//{
		//	name: "Invalid - revoked issuer",
		//	msg: &types.MsgCreateOrUpdatePermissionSession{
		//		Creator:           creator,
		//		Id:                uuid.New().String(),
		//		IssuerPermId:      revokedPermID,
		//		VerifierPermId:    0,
		//		AgentPermId:       agentPermID,
		//		WalletAgentPermId: walletAgentPermID,
		//	},
		//	expectErr:  true,
		//	errMessage: "issuer perm is revoked or terminated",
		//},
		{
			name: "Invalid - agent perm not found",
			msg: &types.MsgCreateOrUpdatePermissionSession{
				Creator:           creator,
				Id:                uuid.New().String(),
				IssuerPermId:      issuerPermID,
				VerifierPermId:    0,
				AgentPermId:       9999,
				WalletAgentPermId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "agent permission not found",
		},
		//{
		//	name: "Invalid - agent not HOLDER type",
		//	msg: &types.MsgCreateOrUpdatePermissionSession{
		//		Creator:           creator,
		//		Id:                uuid.New().String(),
		//		IssuerPermId:      issuerPermID,
		//		VerifierPermId:    0,
		//		AgentPermId:       issuerPermID, // Not HOLDER type
		//		WalletAgentPermId: walletAgentPermID,
		//	},
		//	expectErr:  true,
		//	errMessage: "agent permission must be HOLDER type",
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CreateOrUpdatePermissionSession(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tc.msg.Id, resp.Id)

				// Verify session was created/updated
				session, err := k.PermissionSession.Get(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.AgentPermId, session.AgentPermId)
				require.Equal(t, tc.msg.Creator, session.Controller)

				// Check that the session contains appropriate authorization
				foundAuthz := false
				for _, authz := range session.Authz {
					if authz.ExecutorPermId == tc.msg.IssuerPermId &&
						authz.BeneficiaryPermId == tc.msg.VerifierPermId &&
						authz.WalletAgentPermId == tc.msg.WalletAgentPermId {
						foundAuthz = true
						break
					}
				}
				require.True(t, foundAuthz, "Session doesn't contain the expected authorization")
			}
		})
	}
}

// TestDiscountApplicationInFeeCalculation tests that discounts are correctly applied when calculating fees
func TestDiscountApplicationInFeeCalculation(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create validator perm (ISSUER_GRANTOR) with issuance fees
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		IssuanceFees:  100, // 100 trust units
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create ISSUER perm with discount set (per Issue #94: use discount instead of exemption)
	issuerPerm := types.Permission{
		SchemaId:            1,
		Type:                types.PermissionType_ISSUER,
		Authority:             creator,
		Created:             &now,
		CreatedBy:           creator,
		Extended:            &now,
		ExtendedBy:          creator,
		Modified:            &now,
		Country:             "US",
		ValidatorPermId:     validatorPermID,
		VpState:             types.ValidationState_VALIDATED,
		IssuanceFeeDiscount: 5000, // 50% discount
		EffectiveFrom:       &pastTime,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Create agent perm (HOLDER type)
	agentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID,
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	walletAgentPermID := agentPermID // Use same for simplicity

	t.Run("Discount applied to beneficiary fees", func(t *testing.T) {
		// When creating a session with issuerPermID:
		// 1. Sum fees from found_perm_set (validatorPerm with IssuanceFees=100)
		// 2. Apply exemption from issuerPerm: beneficiary_fees = 100 * (1 - 0.5) = 50
		// Expected: beneficiary_fees = 50

		msg := &types.MsgCreateOrUpdatePermissionSession{
			Creator:           creator,
			Id:                uuid.New().String(),
			IssuerPermId:      issuerPermID,
			VerifierPermId:    0,
			AgentPermId:       agentPermID,
			WalletAgentPermId: walletAgentPermID,
		}

		resp, err := ms.CreateOrUpdatePermissionSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, msg.Id, resp.Id)
	})

	t.Run("Discount applied in execution", func(t *testing.T) {
		// Create another issuer perm with different discount
		issuerPerm2 := types.Permission{
			SchemaId:            1,
			Type:                types.PermissionType_ISSUER,
			Authority:             creator,
			Created:             &now,
			CreatedBy:           creator,
			Extended:            &now,
			ExtendedBy:          creator,
			Modified:            &now,
			Country:             "US",
			ValidatorPermId:     validatorPermID,
			VpState:             types.ValidationState_VALIDATED,
			IssuanceFeeDiscount: 3000, // 30% discount
			EffectiveFrom:       &pastTime,
		}
		issuerPerm2ID, err := k.CreatePermission(sdkCtx, issuerPerm2)
		require.NoError(t, err)

		// Expected: fees from validatorPerm (100) * (1 - 0.3) = 70
		msg := &types.MsgCreateOrUpdatePermissionSession{
			Creator:           creator,
			Id:                uuid.New().String(),
			IssuerPermId:      issuerPerm2ID,
			VerifierPermId:    0,
			AgentPermId:       agentPermID,
			WalletAgentPermId: walletAgentPermID,
		}

		resp, err := ms.CreateOrUpdatePermissionSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("Multiple discounts applied", func(t *testing.T) {
		// Create validator with discount
		validatorWithDiscount := types.Permission{
			SchemaId:            1,
			Type:                types.PermissionType_ISSUER_GRANTOR,
			Authority:             creator,
			Created:             &now,
			CreatedBy:           creator,
			Extended:            &now,
			ExtendedBy:          creator,
			Modified:            &now,
			Country:             "US",
			VpState:             types.ValidationState_VALIDATED,
			IssuanceFees:        200,  // 200 trust units
			IssuanceFeeDiscount: 2000, // 20% discount
			EffectiveFrom:       &pastTime,
		}
		validatorWithDiscountID, err := k.CreatePermission(sdkCtx, validatorWithDiscount)
		require.NoError(t, err)

		// Create issuer with discount (per Issue #94: use discount instead of exemption)
		issuerWithDiscount := types.Permission{
			SchemaId:            1,
			Type:                types.PermissionType_ISSUER,
			Authority:             creator,
			Created:             &now,
			CreatedBy:           creator,
			Extended:            &now,
			ExtendedBy:          creator,
			Modified:            &now,
			Country:             "US",
			ValidatorPermId:     validatorWithDiscountID,
			VpState:             types.ValidationState_VALIDATED,
			IssuanceFeeDiscount: 3000, // 30% discount
			EffectiveFrom:       &pastTime,
		}
		issuerWithDiscountID, err := k.CreatePermission(sdkCtx, issuerWithDiscount)
		require.NoError(t, err)

		require.NoError(t, err)

		// Expected calculation:
		// 1. Apply issuer discount: 200 * (1 - 0.3) = 140
		// Final beneficiary_fees = 140

		msg := &types.MsgCreateOrUpdatePermissionSession{
			Creator:           creator,
			Id:                uuid.New().String(),
			IssuerPermId:      issuerWithDiscountID,
			VerifierPermId:    0,
			AgentPermId:       agentPermID,
			WalletAgentPermId: walletAgentPermID,
		}

		resp, err := ms.CreateOrUpdatePermissionSession(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestGetPermissionByID tests the GetPermissionByID function
func TestGetPermissionByID(t *testing.T) {
	k, _, _, ctx, _ := keepertest.PermissionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Create a test perm
	testPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_ISSUER,
		Authority:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATED,
	}
	permID, err := k.CreatePermission(sdkCtx, testPerm)
	require.NoError(t, err)

	// Test getting the perm
	retrievedPerm, err := k.GetPermissionByID(sdkCtx, permID)
	require.NoError(t, err, "GetPermissionByID should not return an error for a valid ID")
	require.Equal(t, permID, retrievedPerm.Id, "Permission ID should match")
	require.Equal(t, testPerm.SchemaId, retrievedPerm.SchemaId, "Schema ID should match")
	require.Equal(t, testPerm.Type, retrievedPerm.Type, "Type should match")
	require.Equal(t, testPerm.Authority, retrievedPerm.Authority, "Grantee should match")
	require.Equal(t, testPerm.Country, retrievedPerm.Country, "Country should match")

	// Test getting a non-existent perm
	_, err = k.GetPermissionByID(sdkCtx, 9999)
	require.Error(t, err, "GetPermissionByID should return an error for an invalid ID")
}

// TestCreateAndUpdatePermission tests the CreatePermission and UpdatePermission functions
func TestCreateAndUpdatePermission(t *testing.T) {
	k, _, _, ctx, _ := keepertest.PermissionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Test CreatePermission
	testPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_ISSUER,
		Authority:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATED,
	}

	permID, err := k.CreatePermission(sdkCtx, testPerm)
	require.NoError(t, err, "CreatePermission should not return an error")
	require.Greater(t, permID, uint64(0), "Permission ID should be greater than 0")

	// Retrieve the created perm
	retrievedPerm, err := k.GetPermissionByID(sdkCtx, permID)
	require.NoError(t, err)
	require.Equal(t, permID, retrievedPerm.Id, "Created perm ID should match")
	require.Equal(t, testPerm.SchemaId, retrievedPerm.SchemaId, "Created perm schema ID should match")

	// Test UpdatePermission
	updatedCountry := "FR"
	retrievedPerm.Country = updatedCountry
	futureTime := now.Add(24 * time.Hour)
	retrievedPerm.EffectiveUntil = &futureTime

	err = k.UpdatePermission(sdkCtx, retrievedPerm)
	require.NoError(t, err, "UpdatePermission should not return an error")

	// Retrieve the updated perm
	updatedPerm, err := k.GetPermissionByID(sdkCtx, permID)
	require.NoError(t, err)
	require.Equal(t, updatedCountry, updatedPerm.Country, "Country should be updated")
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
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create several permissions for testing
	// Trust Registry perm
	trustPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ECOSYSTEM,
		Did:           validDid,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	trustPermID, err := k.CreatePermission(sdkCtx, trustPerm)
	require.NoError(t, err)

	// Issuer perm
	issuerPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Did:             validDid,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: trustPermID,
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Verifier perm
	verifierPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_VERIFIER,
		Did:             validDid,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "FR", // Different country
		ValidatorPermId: trustPermID,
		VpState:         types.ValidationState_VALIDATED,
		EffectiveFrom:   &pastTime,
	}
	verifierPermID, err := k.CreatePermission(sdkCtx, verifierPerm)

	require.NoError(t, err)

	// Create a session for testing
	sessionID := uuid.New().String()
	session := types.PermissionSession{
		Id:          sessionID,
		Controller:  creator,
		AgentPermId: issuerPermID, // Using issuer as agent for simplicity in test
		Created:     &now,
		Modified:    &now,
		Authz: []*types.SessionAuthz{
			{
				ExecutorPermId:    issuerPermID,
				BeneficiaryPermId: verifierPermID,
			},
		},
	}
	err = k.PermissionSession.Set(sdkCtx, sessionID, session)
	require.NoError(t, err)

	// Test GetPermission query
	getPermReq := &types.QueryGetPermissionRequest{
		Id: issuerPermID,
	}
	getPermResp, err := k.GetPermission(ctx, getPermReq)
	require.NoError(t, err)
	require.NotNil(t, getPermResp)
	require.Equal(t, issuerPermID, getPermResp.Permission.Id)
	require.Equal(t, validDid, getPermResp.Permission.Did)

	// Test ListPermissions query
	listPermReq := &types.QueryListPermissionsRequest{
		ResponseMaxSize: 10,
	}
	listPermResp, err := k.ListPermissions(ctx, listPermReq)
	require.NoError(t, err)
	require.NotNil(t, listPermResp)
	require.GreaterOrEqual(t, len(listPermResp.Permissions), 3) // At least the 3 we created

	// Test GetPermissionSession query
	getSessionReq := &types.QueryGetPermissionSessionRequest{
		Id: sessionID,
	}
	getSessionResp, err := k.GetPermissionSession(ctx, getSessionReq)
	require.NoError(t, err)
	require.NotNil(t, getSessionResp)
	require.Equal(t, sessionID, getSessionResp.Session.Id)
	require.Equal(t, creator, getSessionResp.Session.Controller)

	// Test ListPermissionSessions query
	listSessionsReq := &types.QueryListPermissionSessionsRequest{
		ResponseMaxSize: 10,
	}
	listSessionsResp, err := k.ListPermissionSessions(ctx, listSessionsReq)
	require.NoError(t, err)
	require.NotNil(t, listSessionsResp)
	require.GreaterOrEqual(t, len(listSessionsResp.Sessions), 1) // At least the one we created

	// Test FindPermissionsWithDID query
	findPermDIDReq := &types.QueryFindPermissionsWithDIDRequest{
		Did:      validDid,
		Type:     uint32(types.PermissionType_ISSUER),
		SchemaId: 1,
		Country:  "US",
	}
	findPermDIDResp, err := k.FindPermissionsWithDID(ctx, findPermDIDReq)
	require.NoError(t, err)
	require.NotNil(t, findPermDIDResp)
	require.Equal(t, 1, len(findPermDIDResp.Permissions)) // Should find only the issuer perm
	require.Equal(t, issuerPermID, findPermDIDResp.Permissions[0].Id)

	// Test FindBeneficiaries query
	findBenefReq := &types.QueryFindBeneficiariesRequest{
		IssuerPermId:   issuerPermID,
		VerifierPermId: verifierPermID,
	}
	findBenefResp, err := k.FindBeneficiaries(ctx, findBenefReq)
	require.NoError(t, err)
	require.NotNil(t, findBenefResp)
	require.GreaterOrEqual(t, len(findBenefResp.Permissions), 1) // Should find the trust perm at minimum

	// Find the trust perm in the response
	foundTrustPerm := false
	for _, perm := range findBenefResp.Permissions {
		if perm.Id == trustPermID {
			foundTrustPerm = true
			break
		}
	}
	require.True(t, foundTrustPerm, "Trust registry perm should be in beneficiaries")
}

func TestSlashPermissionTrustDeposit(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	ecosystemAddr := sdk.AccAddress([]byte("test_ecosystem")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create ecosystem perm
	ecosystemPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ECOSYSTEM,
		Authority:       ecosystemAddr,
		Created:       &now,
		CreatedBy:     ecosystemAddr,
		Extended:      &now,
		ExtendedBy:    ecosystemAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       validatorAddr,
		Created:       &now,
		CreatedBy:     validatorAddr,
		Extended:      &now,
		ExtendedBy:    validatorAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create applicant perm with deposit
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		Deposit:         1000, // Set initial deposit
		EffectiveFrom:   &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)

	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgSlashPermissionTrustDeposit
		expectErr  bool
		errMessage string
	}{
		//{
		//	name: "Valid slash by validator",
		//	msg: &types.MsgSlashPermissionTrustDeposit{
		//		Creator: validatorAddr,
		//		Id:      applicantPermID,
		//		Amount:  500,
		//	},
		//	expectErr: false,
		//},
		//{
		//	name: "Valid slash by ecosystem controller",
		//	msg: &types.MsgSlashPermissionTrustDeposit{
		//		Creator: ecosystemAddr,
		//		Id:      applicantPermID,
		//		Amount:  300,
		//	},
		//	expectErr: false,
		//},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgSlashPermissionTrustDeposit{
				Creator: validatorAddr,
				Id:      9999,
				Amount:  100,
			},
			expectErr:  true,
			errMessage: "permission not found",
		},
		{
			name: "Invalid - amount exceeds deposit",
			msg: &types.MsgSlashPermissionTrustDeposit{
				Creator: validatorAddr,
				Id:      applicantPermID,
				Amount:  2000, // More than available deposit
			},
			expectErr:  true,
			errMessage: "amount exceeds available deposit",
		},
		{
			name: "Invalid - unauthorized slasher",
			msg: &types.MsgSlashPermissionTrustDeposit{
				Creator: sdk.AccAddress([]byte("unauthorized")).String(),
				Id:      applicantPermID,
				Amount:  100,
			},
			expectErr:  true,
			errMessage: "creator does not have authority to slash this perm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SlashPermissionTrustDeposit(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm was updated correctly
				perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, perm.Slashed)
				require.Equal(t, tc.msg.Creator, perm.SlashedBy)
				require.Equal(t, tc.msg.Amount, perm.SlashedDeposit)
				require.Equal(t, applicantPerm.Deposit-tc.msg.Amount, perm.Deposit)
			}
		})
	}
}

func TestRepayPermissionSlashedTrustDeposit(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	ecosystemAddr := sdk.AccAddress([]byte("test_ecosystem")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()

	pastTime := now.Add(-1 * time.Hour) // Set effective_from to past relative to block time to make it ACTIVE

	// Create ecosystem perm
	ecosystemPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ECOSYSTEM,
		Authority:       ecosystemAddr,
		Created:       &now,
		CreatedBy:     ecosystemAddr,
		Extended:      &now,
		ExtendedBy:    ecosystemAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       validatorAddr,
		Created:       &now,
		CreatedBy:     validatorAddr,
		Extended:      &now,
		ExtendedBy:    validatorAddr,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create applicant perm with initial deposit
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_ISSUER,
		Authority:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATED,
		Deposit:         1000, // Initial deposit
		EffectiveFrom:   &pastTime,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)

	require.NoError(t, err)

	// First slash the perm
	slashMsg := &types.MsgSlashPermissionTrustDeposit{
		Creator: validatorAddr,
		Id:      applicantPermID,
		Amount:  500, // Slash half of the deposit
	}
	_, err = ms.SlashPermissionTrustDeposit(ctx, slashMsg)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgRepayPermissionSlashedTrustDeposit
		expectErr  bool
		errMessage string
	}{
		//{
		//	name: "Valid repayment",
		//	msg: &types.MsgRepayPermissionSlashedTrustDeposit{
		//		Creator: creator,
		//		Id:      applicantPermID,
		//	},
		//	expectErr: false,
		//},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgRepayPermissionSlashedTrustDeposit{
				Creator: creator,
				Id:      9999,
			},
			expectErr:  true,
			errMessage: "perm not found",
		},
		{
			name: "Invalid - no slashed deposit to repay",
			msg: &types.MsgRepayPermissionSlashedTrustDeposit{
				Creator: creator,
				Id:      validatorPermID, // No slashed deposit
			},
			expectErr:  true,
			errMessage: "no slashed deposit to repay",
		},
		//{
		//	name: "Invalid - already fully repaid",
		//	msg: &types.MsgRepayPermissionSlashedTrustDeposit{
		//		Creator: creator,
		//		Id:      applicantPermID,
		//	},
		//	expectErr:  true,
		//	errMessage: "slashed deposit already fully repaid",
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RepayPermissionSlashedTrustDeposit(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm was updated correctly
				perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, uint64(0), perm.SlashedDeposit) // Slashed deposit should be 0 after repayment
				require.Equal(t, uint64(1000), perm.Deposit)     // Original deposit should be restored
			}
		})
	}
}

func TestCreatePermission(t *testing.T) {
	k, ms, mockCsKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry and store its ID
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema with OPEN perm management modes
	mockCsKeeper.UpdateMockCredentialSchema(1,
		trID,
		cstypes.CredentialSchemaPermManagementMode_OPEN,
		cstypes.CredentialSchemaPermManagementMode_OPEN)

	now := time.Now()
	futureTime := now.Add(24 * time.Hour)

	// Create an ecosystem perm first (required for validation)
	ecosystemPerm := types.Permission{
		SchemaId:  1,
		Type:      types.PermissionType_ECOSYSTEM,
		Did:       validDid,
		Authority:   creator,
		Created:   &now,
		CreatedBy: creator,
		Modified:  &now,
		Country:   "US",
		VpState:   types.ValidationState_VALIDATED,
	}
	ecosystemPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgCreatePermission
		isValid bool
		errMsg  string
	}{
		{
			name: "Valid Issuer Permission",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         1,
				Type:             types.PermissionType_ISSUER,
				Did:              validDid,
				Country:          "US",
				EffectiveFrom:    &now,
				EffectiveUntil:   &futureTime,
				VerificationFees: 100,
			},
			isValid: true,
		},
		//{
		//	name: "Valid Verifier Permission",
		//	msg: &types.MsgCreatePermission{
		//		Creator:          creator,
		//		SchemaId:         1,
		//		Type:             types.PermissionType_VERIFIER,
		//		Did:              validDid,
		//		Country:          "US",
		//		EffectiveFrom:    &now,
		//		EffectiveUntil:   &futureTime,
		//		VerificationFees: 100,
		//	},
		//	isValid: true,
		//},
		{
			name: "Invalid Schema ID",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         999, // Non-existent schema
				Type:             types.PermissionType_ISSUER,
				Did:              validDid,
				Country:          "US",
				VerificationFees: 100,
			},
			isValid: false,
			errMsg:  "credential schema not found",
		},
		//{
		//	name: "Invalid Permission Type",
		//	msg: &types.MsgCreatePermission{
		//		Creator:          creator,
		//		SchemaId:         1,
		//		Type:             types.PermissionType_UNSPECIFIED,
		//		Did:              validDid,
		//		Country:          "US",
		//		VerificationFees: 100,
		//	},
		//	isValid: false,
		//	errMsg:  "type must be ISSUER or VERIFIER",
		//},
		{
			name: "Invalid Country Code",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         1,
				Type:             types.PermissionType_ISSUER,
				Did:              validDid,
				Country:          "INVALID",
				VerificationFees: 100,
			},
			isValid: false,
			errMsg:  "invalid country code format",
		},
		{
			name: "Invalid Effective Dates",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         1,
				Type:             types.PermissionType_ISSUER,
				Did:              validDid,
				Country:          "US",
				EffectiveFrom:    &futureTime,
				EffectiveUntil:   &now, // Before effective_from
				VerificationFees: 100,
			},
			isValid: false,
			errMsg:  "effective_until must be greater than effective_from",
		},
	}

	var expectedID uint64 = 2 // Start from 2 since ecosystem perm is 1

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CreatePermission(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify ID was auto-generated correctly
				require.Equal(t, expectedID, resp.Id)

				// Get the created perm
				perm, err := k.GetPermissionByID(sdkCtx, resp.Id)
				require.NoError(t, err)

				// Verify all fields are set correctly
				require.Equal(t, tc.msg.SchemaId, perm.SchemaId)
				require.Equal(t, tc.msg.Type, perm.Type)
				require.Equal(t, tc.msg.Did, perm.Did)
				require.Equal(t, tc.msg.Creator, perm.Authority)
				require.Equal(t, tc.msg.Country, perm.Country)
				require.Equal(t, tc.msg.VerificationFees, perm.VerificationFees)
				require.Equal(t, ecosystemPermID, perm.ValidatorPermId)
				//require.Equal(t, types.ValidationState_VALIDATED, perm.VpState)

				// Verify time fields if set
				if tc.msg.EffectiveFrom != nil {
					require.Equal(t, tc.msg.EffectiveFrom.Unix(), perm.EffectiveFrom.Unix())
				}
				if tc.msg.EffectiveUntil != nil {
					require.Equal(t, tc.msg.EffectiveUntil.Unix(), perm.EffectiveUntil.Unix())
				}

				// Verify auto-populated fields
				require.NotNil(t, perm.Created)
				require.NotNil(t, perm.Modified)
				require.Equal(t, tc.msg.Creator, perm.CreatedBy)

				expectedID++ // Increment expected ID for next valid creation
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			}
		})
	}
}

// =============================================================================
// ISSUE #191: CreateRootPermission - effective_from MUST be set
// =============================================================================
// This test validates that CreateRootPermission requires effective_from to be set
// and it must be in the future. Per spec [MOD-PERM-MSG-7-2-1]:
// - effective_from is mandatory
// - effective_from must be in the future

func TestCreateRootPermission_EffectiveFromRequired(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	validDid := "did:example:123456789abcdefghi"
	creator := sdk.AccAddress([]byte("test_creator")).String()

	// Create trust registry where creator is the controller
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create credential schema linked to the trust registry
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	futureTime := now.Add(1 * time.Hour)
	pastTime := now.Add(-1 * time.Hour)
	farFutureTime := now.Add(24 * time.Hour)

	testCases := []struct {
		name      string
		msg       *types.MsgCreateRootPermission
		expectErr bool
		errMsg    string
	}{
		{
			// Issue #191: Test that nil effective_from is rejected
			name: "Issue #191: Reject nil effective_from - mandatory field",
			msg: &types.MsgCreateRootPermission{
				Creator:          creator,
				SchemaId:         1,
				Did:              validDid,
				EffectiveFrom:    nil, // NIL - should be rejected
				EffectiveUntil:   nil,
				ValidationFees:   0,
				IssuanceFees:     0,
				VerificationFees: 0,
			},
			expectErr: true,
			errMsg:    "effective_from is required",
		},
		{
			// Issue #191: Test that past effective_from is rejected
			name: "Issue #191: Reject past effective_from - must be in the future",
			msg: &types.MsgCreateRootPermission{
				Creator:          creator,
				SchemaId:         1,
				Did:              validDid,
				EffectiveFrom:    &pastTime, // PAST - should be rejected
				EffectiveUntil:   nil,
				ValidationFees:   0,
				IssuanceFees:     0,
				VerificationFees: 0,
			},
			expectErr: true,
			errMsg:    "effective_from must be in the future",
		},
		{
			// Issue #191: Test that current time (now) is rejected
			name: "Issue #191: Reject effective_from equal to now - must be strictly in the future",
			msg: &types.MsgCreateRootPermission{
				Creator:          creator,
				SchemaId:         1,
				Did:              validDid,
				EffectiveFrom:    &now, // EQUAL TO NOW - should be rejected (not strictly in future)
				EffectiveUntil:   nil,
				ValidationFees:   0,
				IssuanceFees:     0,
				VerificationFees: 0,
			},
			expectErr: true,
			errMsg:    "effective_from must be in the future",
		},
		{
			// Issue #191: Test that future effective_from is accepted
			name: "Issue #191: Accept future effective_from - valid case",
			msg: &types.MsgCreateRootPermission{
				Creator:          creator,
				SchemaId:         1,
				Did:              validDid,
				EffectiveFrom:    &futureTime, // FUTURE - should be accepted
				EffectiveUntil:   &farFutureTime,
				ValidationFees:   0,
				IssuanceFees:     0,
				VerificationFees: 0,
			},
			expectErr: false,
			errMsg:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CreateRootPermission(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify the permission was created with correct effective_from
				perm, err := k.GetPermissionByID(sdkCtx, resp.Id)
				require.NoError(t, err)
				require.NotNil(t, perm.EffectiveFrom)
				require.Equal(t, tc.msg.EffectiveFrom.Unix(), perm.EffectiveFrom.Unix())
			}
		})
	}
}

// =============================================================================
// ISSUE #193: StartPermissionVP - Validator permission must be ACTIVE
// =============================================================================
// This test validates that StartPermissionVP requires the validator permission
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
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)     // In the past - for ACTIVE permissions
	futureTime := now.Add(1 * time.Hour)    // In the future - for FUTURE/INACTIVE permissions
	expiredTime := now.Add(-24 * time.Hour) // Far in the past - for EXPIRED permissions

	// Create an ACTIVE validator permission (valid case for comparison)
	activeValidatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // In the past = ACTIVE
	}
	activeValidatorPermID, err := k.CreatePermission(sdkCtx, activeValidatorPerm)
	require.NoError(t, err)

	// Issue #193: Create a validator permission with NO effective_from (INACTIVE)
	inactiveValidatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: nil, // NULL effective_from = INACTIVE
	}
	inactiveValidatorPermID, err := k.CreatePermission(sdkCtx, inactiveValidatorPerm)
	require.NoError(t, err)

	// Issue #193: Create a validator permission with FUTURE effective_from
	futureValidatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &futureTime, // Future effective_from = not yet ACTIVE
	}
	futureValidatorPermID, err := k.CreatePermission(sdkCtx, futureValidatorPerm)
	require.NoError(t, err)

	// Issue #193: Create an EXPIRED validator permission
	expiredValidatorPerm := types.Permission{
		SchemaId:       1,
		Type:           types.PermissionType_ISSUER_GRANTOR,
		Authority:        creator,
		Created:        &now,
		CreatedBy:      creator,
		Extended:       &now,
		ExtendedBy:     creator,
		Modified:       &now,
		Country:        "US",
		VpState:        types.ValidationState_VALIDATED,
		EffectiveFrom:  &expiredTime,
		EffectiveUntil: &pastTime, // Already expired
	}
	expiredValidatorPermID, err := k.CreatePermission(sdkCtx, expiredValidatorPerm)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		msg       *types.MsgStartPermissionVP
		expectErr bool
		errMsg    string
	}{
		{
			// Baseline: Active validator should work
			name: "Issue #193: Accept ACTIVE validator - valid case",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: activeValidatorPermID,
				Did:             validDid,
			},
			expectErr: false,
			errMsg:    "",
		},
		{
			// Issue #193: Validator with null effective_from should be rejected
			name: "Issue #193: Reject INACTIVE validator - effective_from is null",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: inactiveValidatorPermID,
				Did:             validDid,
			},
			expectErr: true,
			errMsg:    "validator perm is not valid",
		},
		{
			// Issue #193: Validator with future effective_from should be rejected
			name: "Issue #193: Reject FUTURE validator - effective_from is in the future",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: futureValidatorPermID,
				Did:             validDid,
			},
			expectErr: true,
			errMsg:    "validator perm is not valid",
		},
		{
			// Issue #193: Expired validator should be rejected
			name: "Issue #193: Reject EXPIRED validator - effective_until has passed",
			msg: &types.MsgStartPermissionVP{
				Authority:       creator,
				Operator:        creator,
				Type:            types.PermissionType_ISSUER,
				ValidatorPermId: expiredValidatorPermID,
				Did:             validDid,
			},
			expectErr: true,
			errMsg:    "validator perm is not valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.StartPermissionVP(ctx, tc.msg)

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
// ISSUE #196: RevokePermission - Allow revoking not-yet-active permissions
// =============================================================================
// This test validates that RevokePermission allows revoking permissions that
// are not yet active (e.g., effective_from is in the future or null).
// Per spec, no IsValidPermission check is required for revocation.

func TestRevokePermission_AllowNotYetActivePermissions(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)

	// Create an ACTIVE permission (for comparison)
	activePerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime, // ACTIVE
	}
	activePermID, err := k.CreatePermission(sdkCtx, activePerm)
	require.NoError(t, err)

	// Issue #196: Create a permission with FUTURE effective_from (not yet active)
	futurePerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &futureTime, // FUTURE - not yet active
	}
	futurePermID, err := k.CreatePermission(sdkCtx, futurePerm)
	require.NoError(t, err)

	// Issue #196: Create a permission with NULL effective_from (inactive)
	inactivePerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:       creator,
		Created:       &now,
		CreatedBy:     creator,
		Extended:      &now,
		ExtendedBy:    creator,
		Modified:      &now,
		Country:       "US",
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: nil, // INACTIVE - no effective_from
	}
	inactivePermID, err := k.CreatePermission(sdkCtx, inactivePerm)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		msg       *types.MsgRevokePermission
		expectErr bool
		errMsg    string
	}{
		{
			// Baseline: Revoking an ACTIVE permission should work
			name: "Issue #196: Revoke ACTIVE permission - valid case",
			msg: &types.MsgRevokePermission{
				Creator: creator, // Grantee can revoke their own permission
				Id:      activePermID,
			},
			expectErr: false,
			errMsg:    "",
		},
		{
			// Issue #196: Revoking a FUTURE permission (not yet active) should work
			name: "Issue #196: Revoke FUTURE permission - not yet active should be allowed",
			msg: &types.MsgRevokePermission{
				Creator: creator,
				Id:      futurePermID,
			},
			expectErr: false, // Should succeed - this is the fix for Issue #196
			errMsg:    "",
		},
		{
			// Issue #196: Revoking an INACTIVE permission (null effective_from) should work
			name: "Issue #196: Revoke INACTIVE permission - null effective_from should be allowed",
			msg: &types.MsgRevokePermission{
				Creator: creator,
				Id:      inactivePermID,
			},
			expectErr: false, // Should succeed - this is the fix for Issue #196
			errMsg:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RevokePermission(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify the permission was revoked
				perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.NotNil(t, perm.Revoked, "Permission should be revoked")
				require.Equal(t, tc.msg.Creator, perm.RevokedBy, "RevokedBy should match creator")
			}
		})
	}
}

// TestStartPermissionVP_OverlapCheck tests [MOD-PERM-MSG-1-2-4]:
// Cannot have 2 active VPs in the same (schema_id, type, validator_perm_id, authority) context.
func TestStartPermissionVP_OverlapCheck(t *testing.T) {
	k, ms, csKeeper, trkKeeper, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:     creator,
		Created:       &now,
		CreatedBy:     creator,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// First VP should succeed
	msg := &types.MsgStartPermissionVP{
		Authority:       creator,
		Operator:        creator,
		Type:            types.PermissionType_ISSUER,
		ValidatorPermId: validatorPermID,
		Did:             validDid,
	}
	resp, err := ms.StartPermissionVP(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Second VP with same (schema_id, type, validator_perm_id, authority) should fail
	t.Run("Duplicate PENDING VP in same context", func(t *testing.T) {
		msg2 := &types.MsgStartPermissionVP{
			Authority:       creator,
			Operator:        creator,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: validatorPermID,
			Did:             "did:example:different-did",
		}
		resp2, err := ms.StartPermissionVP(ctx, msg2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overlap check failed")
		require.Contains(t, err.Error(), "an active validation process already exists")
		require.Nil(t, resp2)
	})

	// Different authority should succeed (no overlap)
	t.Run("Different authority no overlap", func(t *testing.T) {
		otherCreator := sdk.AccAddress([]byte("other_creator")).String()
		msg3 := &types.MsgStartPermissionVP{
			Authority:       otherCreator,
			Operator:        otherCreator,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: validatorPermID,
			Did:             validDid,
		}
		resp3, err := ms.StartPermissionVP(ctx, msg3)
		require.NoError(t, err)
		require.NotNil(t, resp3)
	})

	// Different type should succeed (no overlap)
	t.Run("Different type no overlap", func(t *testing.T) {
		// Need a VERIFIER_GRANTOR validator for VERIFIER type
		csKeeper.UpdateMockCredentialSchema(1, trID,
			cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
			cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

		verifierGrantorPerm := types.Permission{
			SchemaId:      1,
			Type:          types.PermissionType_VERIFIER_GRANTOR,
			Authority:     creator,
			Created:       &now,
			CreatedBy:     creator,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgPermID, err := k.CreatePermission(sdkCtx, verifierGrantorPerm)
		require.NoError(t, err)

		msg4 := &types.MsgStartPermissionVP{
			Authority:       creator,
			Operator:        creator,
			Type:            types.PermissionType_VERIFIER,
			ValidatorPermId: vgPermID,
			Did:             validDid,
		}
		resp4, err := ms.StartPermissionVP(ctx, msg4)
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

	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:     creator,
		Created:       &now,
		CreatedBy:     creator,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	_, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("AUTHZ-CHECK failure blocks StartPermissionVP", func(t *testing.T) {
		delKeeper.ErrToReturn = fmt.Errorf("operator not authorized for authority")
		defer func() { delKeeper.ErrToReturn = nil }()

		msg := &types.MsgStartPermissionVP{
			Authority:       creator,
			Operator:        sdk.AccAddress([]byte("unauthorized_op")).String(),
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: 1,
			Did:             validDid,
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
		require.Contains(t, err.Error(), "operator not authorized")
		require.Nil(t, resp)
	})

	t.Run("AUTHZ-CHECK success allows StartPermissionVP", func(t *testing.T) {
		delKeeper.ErrToReturn = nil

		msg := &types.MsgStartPermissionVP{
			Authority:       creator,
			Operator:        creator,
			Type:            types.PermissionType_ISSUER,
			ValidatorPermId: 1,
			Did:             validDid,
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
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

	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := sdkCtx.BlockTime()
	pastTime := now.Add(-1 * time.Hour)
	validatorPerm := types.Permission{
		SchemaId:      1,
		Type:          types.PermissionType_ISSUER_GRANTOR,
		Authority:     creator,
		Created:       &now,
		CreatedBy:     creator,
		Modified:      &now,
		VpState:       types.ValidationState_VALIDATED,
		EffectiveFrom: &pastTime,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	t.Run("vs_operator fields propagated to stored permission", func(t *testing.T) {
		operator := sdk.AccAddress([]byte("diff_operator_aa")).String()
		msg := &types.MsgStartPermissionVP{
			Authority:              creator,
			Operator:               operator,
			Type:                   types.PermissionType_ISSUER,
			ValidatorPermId:        validatorPermID,
			Did:                    validDid,
			VsOperator:             vsOperator,
			VsOperatorAuthzEnabled: true,
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, validDid, perm.Did, "DID should be stored")
		require.Equal(t, operator, perm.CreatedBy, "CreatedBy should be operator")
		require.Equal(t, creator, perm.Authority, "Authority should be authority")
		require.Equal(t, vsOperator, perm.VsOperator, "VsOperator should be stored")
		require.True(t, perm.VsOperatorAuthzEnabled, "VsOperatorAuthzEnabled should be true")
		require.Equal(t, uint64(1), perm.SchemaId, "SchemaId should be derived from validator perm")
		require.Equal(t, types.ValidationState_PENDING, perm.VpState)
	})

	t.Run("VERIFIER with VERIFIER_GRANTOR validator", func(t *testing.T) {
		vgPerm := types.Permission{
			SchemaId:      1,
			Type:          types.PermissionType_VERIFIER_GRANTOR,
			Authority:     creator,
			Created:       &now,
			CreatedBy:     creator,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		vgPermID, err := k.CreatePermission(sdkCtx, vgPerm)
		require.NoError(t, err)

		verifierCreator := sdk.AccAddress([]byte("verifier_creator")).String()
		msg := &types.MsgStartPermissionVP{
			Authority:       verifierCreator,
			Operator:        verifierCreator,
			Type:            types.PermissionType_VERIFIER,
			ValidatorPermId: vgPermID,
			Did:             "did:example:verifier-did-123",
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_VERIFIER, perm.Type)
		require.Equal(t, vgPermID, perm.ValidatorPermId)
	})

	t.Run("HOLDER with ISSUER validator", func(t *testing.T) {
		// Create ISSUER perm to serve as validator for HOLDER
		issuerPerm := types.Permission{
			SchemaId:      1,
			Type:          types.PermissionType_ISSUER,
			Authority:     creator,
			Created:       &now,
			CreatedBy:     creator,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
		require.NoError(t, err)

		holderCreator := sdk.AccAddress([]byte("holder_creator_a")).String()
		msg := &types.MsgStartPermissionVP{
			Authority:       holderCreator,
			Operator:        holderCreator,
			Type:            types.PermissionType_HOLDER,
			ValidatorPermId: issuerPermID,
			Did:             "did:example:holder-did-456",
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_HOLDER, perm.Type)
		require.Equal(t, issuerPermID, perm.ValidatorPermId)
	})

	t.Run("HOLDER with wrong validator type rejects", func(t *testing.T) {
		holderCreator := sdk.AccAddress([]byte("holder_bad_val_a")).String()
		msg := &types.MsgStartPermissionVP{
			Authority:       holderCreator,
			Operator:        holderCreator,
			Type:            types.PermissionType_HOLDER,
			ValidatorPermId: validatorPermID, // ISSUER_GRANTOR, not ISSUER
			Did:             "did:example:holder-bad-val",
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "holder perm requires ISSUER validator")
		require.Nil(t, resp)
	})

	t.Run("ECOSYSTEM type combination - ISSUER_GRANTOR with ECOSYSTEM validator", func(t *testing.T) {
		// Create schema with ECOSYSTEM mode for issuer
		csKeeper.UpdateMockCredentialSchema(2, trID,
			cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
			cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

		ecosystemPerm := types.Permission{
			SchemaId:      2,
			Type:          types.PermissionType_ECOSYSTEM,
			Authority:     creator,
			Created:       &now,
			CreatedBy:     creator,
			Modified:      &now,
			VpState:       types.ValidationState_VALIDATED,
			EffectiveFrom: &pastTime,
		}
		ecoPermID, err := k.CreatePermission(sdkCtx, ecosystemPerm)
		require.NoError(t, err)

		grantorCreator := sdk.AccAddress([]byte("grantor_eco_crea")).String()
		msg := &types.MsgStartPermissionVP{
			Authority:       grantorCreator,
			Operator:        grantorCreator,
			Type:            types.PermissionType_ISSUER_GRANTOR,
			ValidatorPermId: ecoPermID,
			Did:             "did:example:issuer-grantor-eco",
		}
		resp, err := ms.StartPermissionVP(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		perm, err := k.GetPermissionByID(sdkCtx, resp.PermissionId)
		require.NoError(t, err)
		require.Equal(t, types.PermissionType_ISSUER_GRANTOR, perm.Type)
	})
}
