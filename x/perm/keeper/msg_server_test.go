package keeper_test

import (
	"context"
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
	k, csKeeper, trkKeeper, ctx := keepertest.PermissionKeeper(t)
	return k, keeper.NewMsgServerImpl(k), csKeeper, trkKeeper, ctx
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

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First create a trust registry for our credential schema
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema with specific perm management modes
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Create validator perm (ISSUER_GRANTOR)
	now := time.Now()
	// This should be VALIDATED as it's a prerequisite
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED, // validator must be validated
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create another validator perm (VERIFIER_GRANTOR with different country)
	verifierGrantorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_VERIFIER_GRANTOR,
		Grantee:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "FR", // Different country
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	verifierGrantorPermID, err := k.CreatePermission(sdkCtx, verifierGrantorPerm)
	require.NoError(t, err)

	testCases := []struct {
		name string
		msg  *types.MsgStartPermissionVP
		err  string
	}{
		{
			name: "Valid ISSUER Permission Request",
			msg: &types.MsgStartPermissionVP{
				Creator:         creator,
				Type:            uint32(types.PermissionType_PERMISSION_TYPE_ISSUER),
				ValidatorPermId: validatorPermID,
				Country:         "US",
				Did:             validDid,
			},
			err: "",
		},
		{
			name: "Non-existent Validator Permission",
			msg: &types.MsgStartPermissionVP{
				Creator:         creator,
				Type:            uint32(types.PermissionType_PERMISSION_TYPE_ISSUER),
				ValidatorPermId: 999,
				Country:         "US",
				Did:             validDid,
			},
			err: "validator perm not found",
		},
		//{
		//	name: "Country Mismatch",
		//	msg: &types.MsgStartPermissionVP{
		//		Creator:         creator,
		//		Type:            uint32(types.PermissionType_PERMISSION_TYPE_ISSUER),
		//		ValidatorPermId: validatorPermID,
		//		Country:         "FR", // Different from validator's country
		//		Did:             validDid,
		//	},
		//	err: "perm validation failed: validator perm is not valid: perm country mismatch: perm has US, requested FR does not contain validator perm country mismatch",
		//},
		{
			name: "Invalid Permission Type Combination - ISSUER with wrong validator",
			msg: &types.MsgStartPermissionVP{
				Creator:         creator,
				Type:            uint32(types.PermissionType_PERMISSION_TYPE_ISSUER),
				ValidatorPermId: verifierGrantorPermID, // Wrong validator type
				Country:         "FR",
				Did:             validDid,
			},
			err: "issuer perm requires ISSUER_GRANTOR validator",
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
				require.Equal(t, tc.msg.Type, uint32(perm.Type))
				require.Equal(t, tc.msg.Creator, perm.Grantee)
				require.Equal(t, tc.msg.Country, perm.Country)
				require.Equal(t, tc.msg.ValidatorPermId, perm.ValidatorPermId)
				require.Equal(t, types.ValidationState_VALIDATION_STATE_PENDING, perm.VpState)
				require.NotNil(t, perm.Created)
				require.NotNil(t, perm.Modified)
				require.NotNil(t, perm.VpLastStateChange)
			}
		})
	}
}

func TestRenewPermissionVP(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	creator := sdk.AccAddress([]byte("test_creator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Create validator perm
	now := time.Now()
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       3, // ISSUER_GRANTOR
		Grantee:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdk.UnwrapSDKContext(ctx), validatorPerm)
	require.NoError(t, err)

	// Create applicant perm
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            1, // ISSUER
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
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
				Creator: creator,
				Id:      999,
			},
			err: "perm not found",
		},
		{
			name: "Wrong Creator",
			msg: &types.MsgRenewPermissionVP{
				Creator: sdk.AccAddress([]byte("wrong_creator")).String(),
				Id:      applicantPermID,
			},
			err: "creator is not the perm grantee",
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
				require.Equal(t, types.ValidationState_VALIDATION_STATE_PENDING, perm.VpState)
				require.NotNil(t, perm.VpLastStateChange)
			}
		})
	}
}

func TestSetPermissionVPToValidated(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	otherAddr := sdk.AccAddress([]byte("other_user")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()
	futureTime := now.Add(365 * 24 * time.Hour)

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// 1. Test with new perm (not renewal case)
	t.Run("Valid new perm validation", func(t *testing.T) {
		// Create a new perm in PENDING state
		newPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
			Grantee:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATION_STATE_PENDING,
		}
		newPermID, err := k.CreatePermission(sdkCtx, newPerm)
		require.NoError(t, err)

		// Set perm to validated
		msg := &types.MsgSetPermissionVPToValidated{
			Creator:            validatorAddr,
			Id:                 newPermID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			Country:            "US",
			EffectiveUntil:     &futureTime,
			VpSummaryDigestSri: "sha384-validDigest",
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify perm was updated correctly
		updatedPerm, err := k.GetPermissionByID(sdkCtx, newPermID)
		require.NoError(t, err)
		require.Equal(t, types.ValidationState_VALIDATION_STATE_VALIDATED, updatedPerm.VpState)
		require.Equal(t, msg.ValidationFees, updatedPerm.ValidationFees)
		require.Equal(t, msg.IssuanceFees, updatedPerm.IssuanceFees)
		require.Equal(t, msg.VerificationFees, updatedPerm.VerificationFees)
		require.Equal(t, msg.Country, updatedPerm.Country)
		require.NotNil(t, updatedPerm.EffectiveFrom)
		require.NotNil(t, updatedPerm.EffectiveUntil)
		require.Equal(t, msg.VpSummaryDigestSri, updatedPerm.VpSummaryDigestSri)
	})

	// 2. Test renewal case - perm already has EffectiveFrom
	//t.Run("Renewal perm validation", func(t *testing.T) {
	//	// Create a perm that already has EffectiveFrom set
	//	effectiveFrom := now.Add(-90 * 24 * time.Hour) // 90 days ago
	//	renewalPerm := types.Permission{
	//		SchemaId:         1,
	//		Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
	//		Grantee:          creator,
	//		Created:          &now,
	//		CreatedBy:        creator,
	//		Extended:         &now,
	//		ExtendedBy:       creator,
	//		Modified:         &now,
	//		Country:          "US",
	//		ValidatorPermId:  validatorPermID,
	//		VpState:          types.ValidationState_VALIDATION_STATE_PENDING,
	//		EffectiveFrom:    &effectiveFrom,
	//		ValidationFees:   10,
	//		IssuanceFees:     5,
	//		VerificationFees: 3,
	//	}
	//	renewalPermID, err := k.CreatePermission(sdkCtx, renewalPerm)
	//	require.NoError(t, err)
	//
	//	// Set perm to validated with same fees
	//	msg := &types.MsgSetPermissionVPToValidated{
	//		Creator:          validatorAddr,
	//		Id:               renewalPermID,
	//		ValidationFees:   10,   // Same as existing
	//		IssuanceFees:     5,    // Same as existing
	//		VerificationFees: 3,    // Same as existing
	//		Country:          "US", // Same as existing
	//		EffectiveUntil:   &futureTime,
	//	}
	//
	//	resp, err := ms.SetPermissionVPToValidated(ctx, msg)
	//	require.NoError(t, err)
	//	require.NotNil(t, resp)
	//
	//	// Verify perm was updated correctly
	//	updatedPerm, err := k.GetPermissionByID(sdkCtx, renewalPermID)
	//	require.NoError(t, err)
	//	require.Equal(t, types.ValidationState_VALIDATION_STATE_VALIDATED, updatedPerm.VpState)
	//	require.Equal(t, renewalPerm.ValidationFees, updatedPerm.ValidationFees)
	//	require.Equal(t, renewalPerm.IssuanceFees, updatedPerm.IssuanceFees)
	//	require.Equal(t, renewalPerm.VerificationFees, updatedPerm.VerificationFees)
	//	require.Equal(t, renewalPerm.Country, updatedPerm.Country)
	//	require.Equal(t, effectiveFrom.Unix(), updatedPerm.EffectiveFrom.Unix())
	//	require.NotNil(t, updatedPerm.EffectiveUntil)
	//	require.NotNil(t, updatedPerm.VpExp)
	//})

	// 3. Test validation error - Invalid Permission ID
	t.Run("Invalid Permission ID", func(t *testing.T) {
		msg := &types.MsgSetPermissionVPToValidated{
			Creator: validatorAddr,
			Id:      9999, // Non-existent ID
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
			Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
			Grantee:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED, // Not PENDING
		}
		notPendingPermID, err := k.CreatePermission(sdkCtx, notPendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Creator: validatorAddr,
			Id:      notPendingPermID,
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
			Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
			Grantee:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATION_STATE_PENDING,
		}
		pendingPermID, err := k.CreatePermission(sdkCtx, pendingPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Creator: otherAddr, // Not the validator
			Id:      pendingPermID,
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "account running method must be validator grantee")
		require.Nil(t, resp)
	})

	// 6. Test validation error - HOLDER with digest SRI
	t.Run("HOLDER type with digest SRI", func(t *testing.T) {
		// Create a HOLDER perm in PENDING state
		holderPerm := types.Permission{
			SchemaId:        1,
			Type:            types.PermissionType_PERMISSION_TYPE_HOLDER,
			Grantee:         creator,
			Created:         &now,
			CreatedBy:       creator,
			Extended:        &now,
			ExtendedBy:      creator,
			Modified:        &now,
			Country:         "US",
			ValidatorPermId: validatorPermID,
			VpState:         types.ValidationState_VALIDATION_STATE_PENDING,
		}
		holderPermID, err := k.CreatePermission(sdkCtx, holderPerm)
		require.NoError(t, err)

		msg := &types.MsgSetPermissionVPToValidated{
			Creator:            validatorAddr,
			Id:                 holderPermID,
			ValidationFees:     10,
			IssuanceFees:       5,
			VerificationFees:   3,
			Country:            "US",
			VpSummaryDigestSri: "sha384-someDigest", // Should be empty for HOLDER
		}

		resp, err := ms.SetPermissionVPToValidated(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vp_summary_digest_sri must be null for HOLDER type")
		require.Nil(t, resp)
	})
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
				require.Equal(t, tc.msg.Creator, perm.Grantee)
				require.Equal(t, types.PermissionType_PERMISSION_TYPE_ECOSYSTEM, perm.Type)
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

func TestRequestPermissionVPTermination(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set the block time for the context
	blockTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Use the block time for our permissions creation
	now := sdkCtx.BlockTime()

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a non-HOLDER perm in VALIDATED state (ISSUER type)
	// For testing termination of non-HOLDER type
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create a clearly expired VP perm (ISSUER type) for testing validator termination
	pastTime := now.Add(-30 * 24 * time.Hour) // 30 days in the past
	expiredVpPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER, // Not HOLDER to avoid that rule
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
		VpExp:           &pastTime, // Clearly expired VP
	}
	expiredVpPermID, err := k.CreatePermission(sdkCtx, expiredVpPerm)
	require.NoError(t, err)

	// Create an active HOLDER type perm (not expired)
	futureTime := now.Add(24 * time.Hour)
	holderPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_HOLDER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
		VpExp:           &futureTime, // Not expired
	}
	holderPermID, err := k.CreatePermission(sdkCtx, holderPerm)
	require.NoError(t, err)

	holderPermID2, err := k.CreatePermission(sdkCtx, holderPerm)
	require.NoError(t, err)

	// Create a perm in PENDING state for testing validation error
	pendingPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_PENDING,
	}
	pendingPermID, err := k.CreatePermission(sdkCtx, pendingPerm)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgRequestPermissionVPTermination
		expectErr  bool
		errMessage string
		checkState bool
		expState   types.ValidationState
	}{
		{
			name: "Valid termination request by grantee for non-HOLDER type",
			msg: &types.MsgRequestPermissionVPTermination{
				Creator: creator,
				Id:      applicantPermID,
			},
			expectErr:  false,
			checkState: true,
			expState:   types.ValidationState_VALIDATION_STATE_TERMINATED, // Non-HOLDER -> directly TERMINATED
		},
		{
			name: "Valid termination of expired VP by validator",
			msg: &types.MsgRequestPermissionVPTermination{
				Creator: validatorAddr, // Validator terminating expired VP
				Id:      expiredVpPermID,
			},
			expectErr:  false,
			checkState: true,
			expState:   types.ValidationState_VALIDATION_STATE_TERMINATED, // Expired -> directly TERMINATED
		},
		{
			name: "Valid termination of active HOLDER type - goes to requested state",
			msg: &types.MsgRequestPermissionVPTermination{
				Creator: creator,
				Id:      holderPermID,
			},
			expectErr:  false,
			checkState: true,
			expState:   types.ValidationState_VALIDATION_STATE_TERMINATION_REQUESTED, // HOLDER + not expired -> TERMINATION_REQUESTED
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgRequestPermissionVPTermination{
				Creator: creator,
				Id:      9999,
			},
			expectErr:  true,
			errMessage: "perm not found",
		},
		{
			name: "Invalid - not in VALIDATED state",
			msg: &types.MsgRequestPermissionVPTermination{
				Creator: creator,
				Id:      pendingPermID, // In PENDING state
			},
			expectErr:  true,
			errMessage: "must be in VALIDATED state",
		},
		{
			name: "Invalid - wrong creator trying to terminate active VP",
			msg: &types.MsgRequestPermissionVPTermination{
				Creator: sdk.AccAddress([]byte("wrong_creator")).String(),
				Id:      holderPermID2, // Active HOLDER VP
			},
			expectErr:  true,
			errMessage: "only grantee can terminate active VP",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RequestPermissionVPTermination(ctx, tc.msg)

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

					// For terminated permissions, check additional fields
					if tc.expState == types.ValidationState_VALIDATION_STATE_TERMINATED {
						require.NotNil(t, perm.Terminated)
						require.Equal(t, tc.msg.Creator, perm.TerminatedBy)
					} else if tc.expState == types.ValidationState_VALIDATION_STATE_TERMINATION_REQUESTED {
						require.NotNil(t, perm.VpTermRequested)
					}
				}
			}
		})
	}
}

// TestConfirmPermissionVPTermination tests the ConfirmPermissionVPTermination message server function
func TestConfirmPermissionVPTermination(t *testing.T) {
	k, ms, csKeeper, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Set specific block time for consistent testing
	blockTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	sdkCtx = sdkCtx.WithBlockTime(blockTime)
	ctx = sdk.WrapSDKContext(sdkCtx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()

	// Set default params for testing (including validation_term_requested_timeout_days)
	params := types.DefaultParams()
	err := k.SetParams(sdkCtx, params)
	require.NoError(t, err)

	// Verify timeout days is the default value (7 days)
	defaultTimeoutDays := k.GetParams(sdkCtx).ValidationTermRequestedTimeoutDays
	require.Equal(t, uint64(7), defaultTimeoutDays)

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	// Use the block time for permissions
	now := sdkCtx.BlockTime()

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm with termination requested with recent request time
	// Only 1 hour ago, so timeout (7 days) not reached
	termRequested := now.Add(-1 * time.Hour)
	applicantPerm := types.Permission{
		SchemaId:           1,
		Type:               types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:            creator,
		Created:            &now,
		CreatedBy:          creator,
		Extended:           &now,
		ExtendedBy:         creator,
		Modified:           &now,
		Country:            "US",
		ValidatorPermId:    validatorPermID,
		VpState:            types.ValidationState_VALIDATION_STATE_TERMINATION_REQUESTED,
		VpTermRequested:    &termRequested,
		Deposit:            100, // Add some deposit to test it gets processed
		VpValidatorDeposit: 50,  // Validator deposit
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create a duplicate for testing applicant confirmation before timeout
	applicantPermID2, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create a perm with termination requested with timeout expired
	// 8 days ago (past default 7 day timeout)
	termRequestedTimeout := now.Add(-8 * 24 * time.Hour)
	applicantPermTimeoutExpired := types.Permission{
		SchemaId:           1,
		Type:               types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:            creator,
		Created:            &now,
		CreatedBy:          creator,
		Extended:           &now,
		ExtendedBy:         creator,
		Modified:           &now,
		Country:            "US",
		ValidatorPermId:    validatorPermID,
		VpState:            types.ValidationState_VALIDATION_STATE_TERMINATION_REQUESTED,
		VpTermRequested:    &termRequestedTimeout,
		Deposit:            100,
		VpValidatorDeposit: 50,
	}
	applicantPermTimeoutExpiredID, err := k.CreatePermission(sdkCtx, applicantPermTimeoutExpired)
	applicantPermTimeoutExpiredID2, err := k.CreatePermission(sdkCtx, applicantPermTimeoutExpired)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		msg        *types.MsgConfirmPermissionVPTermination
		expectErr  bool
		errMessage string
	}{
		{
			name: "Valid confirmation by validator before timeout",
			msg: &types.MsgConfirmPermissionVPTermination{
				Creator: validatorAddr,
				Id:      applicantPermID,
			},
			expectErr: false,
		},
		{
			name: "Valid confirmation by applicant after timeout",
			msg: &types.MsgConfirmPermissionVPTermination{
				Creator: creator,
				Id:      applicantPermTimeoutExpiredID,
			},
			expectErr: false,
		},
		{
			name: "Invalid - perm not found",
			msg: &types.MsgConfirmPermissionVPTermination{
				Creator: creator,
				Id:      9999,
			},
			expectErr:  true,
			errMessage: "perm not found",
		},
		{
			name: "Invalid - wrong state",
			msg: &types.MsgConfirmPermissionVPTermination{
				Creator: validatorAddr,
				Id:      validatorPermID, // Not in TERMINATION_REQUESTED state
			},
			expectErr:  true,
			errMessage: "must be in TERMINATION_REQUESTED state",
		},
		{
			name: "Invalid - applicant cannot confirm before timeout",
			msg: &types.MsgConfirmPermissionVPTermination{
				Creator: creator,
				Id:      applicantPermID2, // Should be confirmed by validator before timeout
			},
			expectErr:  true,
			errMessage: "only validator can confirm termination before timeout",
		},
		{
			name: "Invalid - unrelated party attempting confirmation",
			msg: &types.MsgConfirmPermissionVPTermination{
				Creator: sdk.AccAddress([]byte("unrelated_party")).String(),
				Id:      applicantPermTimeoutExpiredID2,
			},
			expectErr:  true,
			errMessage: "only validator or applicant can confirm",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.ConfirmPermissionVPTermination(ctx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMessage)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify perm state was updated correctly
				perm, err := k.GetPermissionByID(sdkCtx, tc.msg.Id)
				require.NoError(t, err)
				require.Equal(t, types.ValidationState_VALIDATION_STATE_TERMINATED, perm.VpState)
				require.NotNil(t, perm.Terminated)
				require.Equal(t, tc.msg.Creator, perm.TerminatedBy)

				// Check that deposits were properly processed
				require.Equal(t, uint64(0), perm.Deposit)

				// If terminated by validator, validator deposit should be returned
				if tc.msg.Creator == validatorAddr {
					require.Equal(t, uint64(0), perm.VpValidatorDeposit)
				}
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
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm in PENDING state that has never been validated (vp_exp is nil)
	// This should transition to TERMINATED when cancelled
	neverValidatedPerm := types.Permission{
		SchemaId:         1,
		Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:          creator,
		Created:          &now,
		CreatedBy:        creator,
		Extended:         &now,
		ExtendedBy:       creator,
		Modified:         &now,
		Country:          "US",
		ValidatorPermId:  validatorPermID,
		VpState:          types.ValidationState_VALIDATION_STATE_PENDING,
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
		Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:          creator,
		Created:          &now,
		CreatedBy:        creator,
		Extended:         &now,
		ExtendedBy:       creator,
		Modified:         &now,
		Country:          "US",
		ValidatorPermId:  validatorPermID,
		VpState:          types.ValidationState_VALIDATION_STATE_PENDING,
		VpExp:            &futureTime, // Has a previous validation
		VpCurrentFees:    100,
		VpCurrentDeposit: 50,
	}
	previouslyValidatedPermID, err := k.CreatePermission(sdkCtx, previouslyValidatedPerm)
	require.NoError(t, err)

	// Create a perm not in PENDING state for testing validation error
	notPendingPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED, // Not in PENDING state
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
			expState:   types.ValidationState_VALIDATION_STATE_TERMINATED,
		},
		{
			name: "Valid cancellation - previously validated",
			msg: &types.MsgCancelPermissionVPLastRequest{
				Creator: creator,
				Id:      previouslyValidatedPermID,
			},
			expectErr:  false,
			checkState: true,
			expState:   types.ValidationState_VALIDATION_STATE_VALIDATED,
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
			errMessage: "creator is not the perm grantee",
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

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm to extend
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  &currentEffectiveUntil,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
		VpExp:           &futureVpExp,
	}
	applicantPermID, err := k.CreatePermission(sdkCtx, applicantPerm)
	require.NoError(t, err)

	// Create a trust registry perm to test direct extension
	trustRegistryPerm := types.Permission{
		SchemaId:       1,
		Type:           types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Grantee:        trustRegistryAddr,
		Created:        &now,
		CreatedBy:      trustRegistryAddr,
		Extended:       &now,
		ExtendedBy:     trustRegistryAddr,
		Modified:       &now,
		EffectiveUntil: &currentEffectiveUntil,
		Country:        "US",
		VpState:        types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	trustRegistryPermID, err := k.CreatePermission(sdkCtx, trustRegistryPerm)
	require.NoError(t, err)

	// Create a separate perm for the "wrong creator" test
	// Use same validator but has a different effective_until date
	wrongCreatorTestPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		EffectiveUntil:  &currentEffectiveUntil, // Same as the regular test
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
		VpExp:           &futureVpExp,
	}
	wrongCreatorTestPermID, err := k.CreatePermission(sdkCtx, wrongCreatorTestPerm)
	require.NoError(t, err)

	newEffectiveUntil := now.Add(60 * 24 * time.Hour)     // 60 days in the future
	pastEffectiveUntil := now.Add(-1 * 24 * time.Hour)    // 1 day in the past
	tooFarEffectiveUntil := now.Add(500 * 24 * time.Hour) // Past VP expiration

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
			errMessage: "perm not found",
		},
		{
			name: "Invalid - effective_until not after current effective_until",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &currentEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be after current effective_until",
		},
		{
			name: "Invalid - effective_until in the past",
			msg: &types.MsgExtendPermission{
				Creator:        validatorAddr,
				Id:             applicantPermID,
				EffectiveUntil: &pastEffectiveUntil,
			},
			expectErr:  true,
			errMessage: "effective_until must be after current effective_until",
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

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create a perm to revoke
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
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
			errMessage: "perm not found",
		},
		{
			name: "Invalid - validator not found",
			msg: &types.MsgRevokePermission{
				Creator: validatorAddr,
				Id:      validatorPermID, // Validator perm has no validator
			},
			expectErr:  true,
			errMessage: "validator perm not found",
		},
		{
			name: "Invalid - wrong creator",
			msg: &types.MsgRevokePermission{
				Creator: creator,
				Id:      applicantPermID,
			},
			expectErr:  true,
			errMessage: "creator is not the validator",
		},
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

	// Create trust registry / validator perm
	trustPerm := types.Permission{
		SchemaId:         1,
		Type:             types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Grantee:          creator,
		Created:          &now,
		CreatedBy:        creator,
		Extended:         &now,
		ExtendedBy:       creator,
		Modified:         &now,
		Country:          "US",
		VpState:          types.ValidationState_VALIDATION_STATE_VALIDATED,
		ValidationFees:   10,
		IssuanceFees:     5,
		VerificationFees: 3,
	}
	trustPermID, err := k.CreatePermission(sdkCtx, trustPerm)
	require.NoError(t, err)

	// Create issuer perm
	issuerPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: trustPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Create verifier perm
	//verifierPerm := types.Permission{
	//	SchemaId:        1,
	//	Type:            types.PermissionType_PERMISSION_TYPE_VERIFIER,
	//	Grantee:         creator,
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
		Type:            types.PermissionType_PERMISSION_TYPE_HOLDER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID, // Issued by the issuer
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	agentPermID, err := k.CreatePermission(sdkCtx, agentPerm)
	require.NoError(t, err)

	// Create wallet agent perm (HOLDER type)
	walletAgentPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_HOLDER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: issuerPermID, // Issued by the issuer
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	walletAgentPermID, err := k.CreatePermission(sdkCtx, walletAgentPerm)
	require.NoError(t, err)

	// Create revoked perm
	//revokedPerm := types.Permission{
	//	SchemaId:        1,
	//	Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
	//	Grantee:         creator,
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
			errMessage: "issuer perm not found",
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
			errMessage: "issuer perm must be ISSUER type",
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
			errMessage: "agent perm not found",
		},
		{
			name: "Invalid - agent not HOLDER type",
			msg: &types.MsgCreateOrUpdatePermissionSession{
				Creator:           creator,
				Id:                uuid.New().String(),
				IssuerPermId:      issuerPermID,
				VerifierPermId:    0,
				AgentPermId:       issuerPermID, // Not HOLDER type
				WalletAgentPermId: walletAgentPermID,
			},
			expectErr:  true,
			errMessage: "agent perm must be HOLDER type",
		},
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

// TestGetPermissionByID tests the GetPermissionByID function
func TestGetPermissionByID(t *testing.T) {
	k, _, _, ctx := keepertest.PermissionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Create a test perm
	testPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	permID, err := k.CreatePermission(sdkCtx, testPerm)
	require.NoError(t, err)

	// Test getting the perm
	retrievedPerm, err := k.GetPermissionByID(sdkCtx, permID)
	require.NoError(t, err, "GetPermissionByID should not return an error for a valid ID")
	require.Equal(t, permID, retrievedPerm.Id, "Permission ID should match")
	require.Equal(t, testPerm.SchemaId, retrievedPerm.SchemaId, "Schema ID should match")
	require.Equal(t, testPerm.Type, retrievedPerm.Type, "Type should match")
	require.Equal(t, testPerm.Grantee, retrievedPerm.Grantee, "Grantee should match")
	require.Equal(t, testPerm.Country, retrievedPerm.Country, "Country should match")

	// Test getting a non-existent perm
	_, err = k.GetPermissionByID(sdkCtx, 9999)
	require.Error(t, err, "GetPermissionByID should return an error for an invalid ID")
}

// TestCreateAndUpdatePermission tests the CreatePermission and UpdatePermission functions
func TestCreateAndUpdatePermission(t *testing.T) {
	k, _, _, ctx := keepertest.PermissionKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	now := time.Now()

	// Test CreatePermission
	testPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
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

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDid := "did:example:123456789abcdefghi"

	// Create a trust registry
	trID := trkKeeper.CreateMockTrustRegistry(creator, validDid)

	// Create mock credential schema
	csKeeper.UpdateMockCredentialSchema(1, trID,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create several permissions for testing
	// Trust Registry perm
	trustPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Did:        validDid,
		Grantee:    creator,
		Created:    &now,
		CreatedBy:  creator,
		Extended:   &now,
		ExtendedBy: creator,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	trustPermID, err := k.CreatePermission(sdkCtx, trustPerm)
	require.NoError(t, err)

	// Issuer perm
	issuerPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Did:             validDid,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: trustPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	issuerPermID, err := k.CreatePermission(sdkCtx, issuerPerm)
	require.NoError(t, err)

	// Verifier perm
	verifierPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_VERIFIER,
		Did:             validDid,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "FR", // Different country
		ValidatorPermId: trustPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
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
		Type:     uint32(types.PermissionType_PERMISSION_TYPE_ISSUER),
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

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	ecosystemAddr := sdk.AccAddress([]byte("test_ecosystem")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create ecosystem perm
	ecosystemPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Grantee:    ecosystemAddr,
		Created:    &now,
		CreatedBy:  ecosystemAddr,
		Extended:   &now,
		ExtendedBy: ecosystemAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	_, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create applicant perm with deposit
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
		Deposit:         1000, // Set initial deposit
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
			errMessage: "perm not found",
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

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validatorAddr := sdk.AccAddress([]byte("test_validator")).String()
	ecosystemAddr := sdk.AccAddress([]byte("test_ecosystem")).String()

	// Create mock credential schema
	csKeeper.CreateMockCredentialSchema(1,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION,
		cstypes.CredentialSchemaPermManagementMode_GRANTOR_VALIDATION)

	now := time.Now()

	// Create ecosystem perm
	ecosystemPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Grantee:    ecosystemAddr,
		Created:    &now,
		CreatedBy:  ecosystemAddr,
		Extended:   &now,
		ExtendedBy: ecosystemAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	_, err := k.CreatePermission(sdkCtx, ecosystemPerm)
	require.NoError(t, err)

	// Create validator perm
	validatorPerm := types.Permission{
		SchemaId:   1,
		Type:       types.PermissionType_PERMISSION_TYPE_ISSUER_GRANTOR,
		Grantee:    validatorAddr,
		Created:    &now,
		CreatedBy:  validatorAddr,
		Extended:   &now,
		ExtendedBy: validatorAddr,
		Modified:   &now,
		Country:    "US",
		VpState:    types.ValidationState_VALIDATION_STATE_VALIDATED,
	}
	validatorPermID, err := k.CreatePermission(sdkCtx, validatorPerm)
	require.NoError(t, err)

	// Create applicant perm with initial deposit
	applicantPerm := types.Permission{
		SchemaId:        1,
		Type:            types.PermissionType_PERMISSION_TYPE_ISSUER,
		Grantee:         creator,
		Created:         &now,
		CreatedBy:       creator,
		Extended:        &now,
		ExtendedBy:      creator,
		Modified:        &now,
		Country:         "US",
		ValidatorPermId: validatorPermID,
		VpState:         types.ValidationState_VALIDATION_STATE_VALIDATED,
		Deposit:         1000, // Initial deposit
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
		Type:      types.PermissionType_PERMISSION_TYPE_ECOSYSTEM,
		Did:       validDid,
		Grantee:   creator,
		Created:   &now,
		CreatedBy: creator,
		Modified:  &now,
		Country:   "US",
		VpState:   types.ValidationState_VALIDATION_STATE_VALIDATED,
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
				Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
				Did:              validDid,
				Country:          "US",
				EffectiveFrom:    &now,
				EffectiveUntil:   &futureTime,
				VerificationFees: 100,
			},
			isValid: true,
		},
		{
			name: "Valid Verifier Permission",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         1,
				Type:             types.PermissionType_PERMISSION_TYPE_VERIFIER,
				Did:              validDid,
				Country:          "US",
				EffectiveFrom:    &now,
				EffectiveUntil:   &futureTime,
				VerificationFees: 100,
			},
			isValid: true,
		},
		{
			name: "Invalid Schema ID",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         999, // Non-existent schema
				Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
				Did:              validDid,
				Country:          "US",
				VerificationFees: 100,
			},
			isValid: false,
			errMsg:  "credential schema not found",
		},
		{
			name: "Invalid Permission Type",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         1,
				Type:             types.PermissionType_PERMISSION_TYPE_UNSPECIFIED,
				Did:              validDid,
				Country:          "US",
				VerificationFees: 100,
			},
			isValid: false,
			errMsg:  "type must be ISSUER or VERIFIER",
		},
		{
			name: "Invalid Country Code",
			msg: &types.MsgCreatePermission{
				Creator:          creator,
				SchemaId:         1,
				Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
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
				Type:             types.PermissionType_PERMISSION_TYPE_ISSUER,
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
				require.Equal(t, tc.msg.Creator, perm.Grantee)
				require.Equal(t, tc.msg.Country, perm.Country)
				require.Equal(t, tc.msg.VerificationFees, perm.VerificationFees)
				require.Equal(t, ecosystemPermID, perm.ValidatorPermId)
				require.Equal(t, types.ValidationState_VALIDATION_STATE_VALIDATED, perm.VpState)

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
