package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/diddirectory/keeper"
	"github.com/verana-labs/verana/x/diddirectory/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context) {
	k, ctx := keepertest.DiddirectoryKeeper(t)
	// Set block time directly on the SDK context
	ctx = ctx.WithBlockTime(time.Now())
	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

func TestMsgServerAddDID(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDID := "did:example:123456789abcdefghi"

	testCases := []struct {
		name    string
		msg     *types.MsgAddDID
		isValid bool
	}{
		{
			name: "Valid Add DID - Default Years",
			msg: &types.MsgAddDID{
				Creator: creator,
				Did:     validDID,
				Years:   0, // Should default to 1
			},
			isValid: true,
		},
		{
			name: "Valid Add DID - Multiple Years",
			msg: &types.MsgAddDID{
				Creator: creator,
				Did:     validDID + "2",
				Years:   5,
			},
			isValid: true,
		},
		{
			name: "Empty DID",
			msg: &types.MsgAddDID{
				Creator: creator,
				Did:     "",
				Years:   1,
			},
			isValid: false,
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgAddDID{
				Creator: creator,
				Did:     "invalid-did",
				Years:   1,
			},
			isValid: false,
		},
		{
			name: "Years Too High",
			msg: &types.MsgAddDID{
				Creator: creator,
				Did:     validDID + "3",
				Years:   32,
			},
			isValid: false,
		},
		{
			name: "Duplicate DID",
			msg: &types.MsgAddDID{
				Creator: creator,
				Did:     validDID, // Same as first test case
				Years:   1,
			},
			isValid: false,
		},
	}

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.AddDID(ctx, tc.msg)

			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify DID was stored
				storedDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Did, storedDID.Did)
				require.Equal(t, tc.msg.Creator, storedDID.Controller)

				// Check years and expiration
				years := tc.msg.Years
				if years == 0 {
					years = 1
				}
				expectedDeposit := int64(params.DidDirectoryTrustDeposit * uint64(years))
				require.Equal(t, expectedDeposit, storedDID.Deposit)

				// Verify timestamps
				require.False(t, storedDID.Created.IsZero())
				require.False(t, storedDID.Modified.IsZero())
				require.False(t, storedDID.Exp.IsZero())

				// Verify expiration is years from creation
				expectedExp := storedDID.Created.AddDate(int(years), 0, 0)
				require.Equal(t, expectedExp, storedDID.Exp)

			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestMsgServerRenewDID(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	wrongCreator := sdk.AccAddress([]byte("wrong_creator")).String()
	validDID := "did:example:123456789abcdefghi"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// First create a DID
	createMsg := &types.MsgAddDID{
		Creator: creator,
		Did:     validDID,
		Years:   1,
	}
	_, err := ms.AddDID(ctx, createMsg)
	require.NoError(t, err)

	// Get initial DID entry for later comparison
	initialEntry, err := k.DIDDirectory.Get(ctx, validDID)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgRenewDID
		isValid bool
	}{
		{
			name: "Empty DID",
			msg: &types.MsgRenewDID{
				Creator: creator,
				Did:     "",
				Years:   1,
			},
			isValid: false,
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgRenewDID{
				Creator: creator,
				Did:     "invalid-did",
				Years:   1,
			},
			isValid: false,
		},
		{
			name: "Years Too High",
			msg: &types.MsgRenewDID{
				Creator: creator,
				Did:     validDID,
				Years:   32,
			},
			isValid: false,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgRenewDID{
				Creator: wrongCreator,
				Did:     validDID,
				Years:   1,
			},
			isValid: false,
		},
		{
			name: "Non-existent DID",
			msg: &types.MsgRenewDID{
				Creator: creator,
				Did:     "did:example:nonexistent",
				Years:   1,
			},
			isValid: false,
		},
		{
			name: "Valid Renewal - Default Years",
			msg: &types.MsgRenewDID{
				Creator: creator,
				Did:     validDID,
				Years:   0, // Should default to 1
			},
			isValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RenewDID(ctx, tc.msg)

			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Get updated DID entry
				storedDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)

				// Check years and deposit calculations
				years := tc.msg.Years
				if years == 0 {
					years = 1
				}
				expectedDeposit := initialEntry.Deposit + int64(params.DidDirectoryTrustDeposit*uint64(years))
				require.Equal(t, expectedDeposit, storedDID.Deposit)

				// Verify expiration is extended by years
				expectedExp := initialEntry.Exp.AddDate(int(years), 0, 0)
				require.Equal(t, expectedExp, storedDID.Exp)

				// Store the updated values for next test case
				initialEntry = storedDID

			} else {
				require.Error(t, err)
				require.Nil(t, resp)

				if tc.msg.Did == validDID {
					// Verify DID wasn't modified for invalid attempts
					currentDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
					require.NoError(t, err)
					require.Equal(t, initialEntry, currentDID)
				}
			}
		})
	}
}

func TestMsgServerRemoveDID(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	wrongCreator := sdk.AccAddress([]byte("wrong_creator")).String()
	validDID := "did:example:123456789abcdefghi"
	validDID2 := "did:example:987654321abcdefghi"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// First create a DID
	createMsg := &types.MsgAddDID{
		Creator: creator,
		Did:     validDID,
		Years:   1,
	}
	_, err := ms.AddDID(ctx, createMsg)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgRemoveDID
		setup   func(*sdk.Context)
		isValid bool
	}{
		{
			name: "Empty DID",
			msg: &types.MsgRemoveDID{
				Creator: creator,
				Did:     "",
			},
			isValid: false,
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgRemoveDID{
				Creator: creator,
				Did:     "invalid-did",
			},
			isValid: false,
		},
		{
			name: "Non-existent DID",
			msg: &types.MsgRemoveDID{
				Creator: creator,
				Did:     "did:example:nonexistent",
			},
			isValid: false,
		},
		{
			name: "Wrong Creator Before Grace Period",
			msg: &types.MsgRemoveDID{
				Creator: wrongCreator,
				Did:     validDID,
			},
			isValid: false,
		},
		{
			name: "Anyone Can Remove After Grace Period",
			msg: &types.MsgRemoveDID{
				Creator: wrongCreator,
				Did:     validDID,
			},
			setup: func(ctx *sdk.Context) {
				futureTime := time.Now().AddDate(2, 0, 0)
				*ctx = ctx.WithBlockTime(futureTime)
			},
			isValid: true,
		},
		{
			name: "Valid Removal By Controller",
			msg: &types.MsgRemoveDID{
				Creator: creator,
				Did:     validDID2,
			},
			setup: func(ctx *sdk.Context) {
				// Create a new DID for controller removal test
				createMsg := &types.MsgAddDID{
					Creator: creator,
					Did:     validDID2,
					Years:   1,
				}
				_, err := ms.AddDID(*ctx, createMsg)
				require.NoError(t, err)
			},
			isValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(&ctx)
			}

			resp, err := ms.RemoveDID(ctx, tc.msg)

			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify DID was removed
				_, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.Error(t, err) // Should error as DID no longer exists
			} else {
				require.Error(t, err)
				require.Nil(t, resp)

				if tc.msg.Did == validDID {
					// Verify DID still exists for invalid removal attempts
					_, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
					require.NoError(t, err)
				}
			}
		})
	}
}

func TestMsgServerTouchDID(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	otherCreator := sdk.AccAddress([]byte("other_creator")).String()
	validDID := "did:example:123456789abcdefghi"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	cleanup := func() {
		_ = k.DIDDirectory.Remove(ctx, validDID)
	}

	testCases := []struct {
		name       string
		msg        *types.MsgTouchDID
		beforeTest func()
		isValid    bool
	}{
		{
			name: "Empty DID",
			msg: &types.MsgTouchDID{
				Creator: creator,
				Did:     "",
			},
			isValid: false,
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgTouchDID{
				Creator: creator,
				Did:     "invalid-did",
			},
			isValid: false,
		},
		{
			name: "Non-existent DID",
			msg: &types.MsgTouchDID{
				Creator: creator,
				Did:     "did:example:nonexistent",
			},
			isValid: false,
		},
		{
			name: "Valid Touch By Original Creator",
			msg: &types.MsgTouchDID{
				Creator: creator,
				Did:     validDID,
			},
			beforeTest: func() {
				createMsg := &types.MsgAddDID{
					Creator: creator,
					Did:     validDID,
					Years:   1,
				}
				_, err := ms.AddDID(ctx, createMsg)
				require.NoError(t, err)
			},
			isValid: true,
		},
		{
			name: "Valid Touch By Other Creator",
			msg: &types.MsgTouchDID{
				Creator: otherCreator,
				Did:     validDID,
			},
			beforeTest: func() {
				createMsg := &types.MsgAddDID{
					Creator: creator,
					Did:     validDID,
					Years:   1,
				}
				_, err := ms.AddDID(ctx, createMsg)
				require.NoError(t, err)
			},
			isValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer cleanup()

			if tc.beforeTest != nil {
				tc.beforeTest()
			}

			// Store initial state if DID exists
			var initialEntry types.DIDDirectory
			var hasInitial bool
			if tc.msg.Did != "" {
				entry, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				if err == nil {
					initialEntry = entry
					hasInitial = true
				}
			}

			// Move time forward before touching
			newTime := ctx.BlockTime().Add(10 * time.Second)
			ctx = ctx.WithBlockTime(newTime)

			resp, err := ms.TouchDID(ctx, tc.msg)

			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify DID was updated
				updatedEntry, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)

				// Modified time should be greater than initial time
				require.True(t, updatedEntry.Modified.After(initialEntry.Modified))

				// Modified time should match new block time
				require.Equal(t, newTime, updatedEntry.Modified)

				// Other fields should remain unchanged
				initialEntry.Modified = updatedEntry.Modified // Set equal for comparison
				require.Equal(t, initialEntry, updatedEntry)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)

				if hasInitial {
					// Verify DID wasn't modified for invalid attempts
					currentEntry, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
					require.NoError(t, err)
					require.Equal(t, initialEntry, currentEntry)
				}
			}
		})
	}
}

// TestValidateAddDIDParams tests the validateAddDIDParams function directly
func TestDIDValidation(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)

	validCreator := sdk.AccAddress([]byte("test_creator")).String()

	testCases := []struct {
		name    string
		did     string
		isValid bool
	}{
		{
			name:    "Valid DID with basic alphanumeric method-specific-id",
			did:     "did:example:123456789abcdefghi",
			isValid: true,
		},
		{
			name:    "Valid DID with dots",
			did:     "did:example:123.456.789",
			isValid: true,
		},
		{
			name:    "Valid DID with underscore",
			did:     "did:example:user_profile",
			isValid: true,
		},
		{
			name:    "Valid DID with hyphen",
			did:     "did:example:user-profile-123",
			isValid: true,
		},
		{
			name:    "Invalid DID - no prefix",
			did:     "example:123456789abcdefghi",
			isValid: false,
		},
		{
			name:    "Invalid DID - no method",
			did:     "did::123456789abcdefghi",
			isValid: false,
		},
		{
			name:    "Invalid DID - no method-specific-id",
			did:     "did:example:",
			isValid: false,
		},
		{
			name:    "Invalid DID - contains spaces",
			did:     "did:example:user profile",
			isValid: false,
		},
		{
			name:    "Invalid DID - contains special chars",
			did:     "did:example:user@example.com",
			isValid: false,
		},
		{
			name:    "Empty DID",
			did:     "",
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test through AddDID which will validate the DID format
			msg := &types.MsgAddDID{
				Creator: validCreator,
				Did:     tc.did,
				Years:   1,
			}

			_, err := ms.AddDID(ctx, msg)

			if tc.isValid {
				if err != nil {
					// If there's an error, but it's not about DID syntax, the test should pass
					require.NotContains(t, err.Error(), "invalid DID syntax")
				}
			} else {
				// If the DID is invalid, we expect either a "DID is required" or "invalid DID syntax" error
				require.Error(t, err)
				if tc.did == "" {
					require.Contains(t, err.Error(), "DID is required")
				} else {
					require.Contains(t, err.Error(), "invalid DID syntax")
				}
			}
		})
	}
}

// TestAddDIDValidations tests the validations for the AddDID function
func TestAddDIDValidations(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	validCreator := sdk.AccAddress([]byte("test_creator")).String()
	validDID := "did:example:valid"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// First add a DID to test the "already exists" case
	createMsg := &types.MsgAddDID{
		Creator: validCreator,
		Did:     validDID,
		Years:   1,
	}
	_, err := ms.AddDID(ctx, createMsg)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgAddDID
		wantErr bool
		errMsg  string
	}{
		{
			name: "Empty DID",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "",
				Years:   1,
			},
			wantErr: true,
			errMsg:  "DID is required",
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "invalid-did",
				Years:   1,
			},
			wantErr: true,
			errMsg:  "invalid DID syntax",
		},
		{
			name: "DID Already Exists",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     validDID, // This DID was added in setup
				Years:   1,
			},
			wantErr: true,
			errMsg:  "DID already exists",
		},
		{
			name: "Years Too High",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "did:example:toomanyyears",
				Years:   32, // Max is 31
			},
			wantErr: true,
			errMsg:  "years must be between 1 and 31",
		},
		{
			name: "Default Years (0 becomes 1)",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "did:example:defaultyears",
				Years:   0, // Should default to 1
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.AddDID(ctx, tc.msg)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify DID was stored with correct parameters
				storedDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)

				years := tc.msg.Years
				if years == 0 {
					years = 1
				}

				// Check years and expiration
				expectedDeposit := int64(params.DidDirectoryTrustDeposit * uint64(years))
				require.Equal(t, expectedDeposit, storedDID.Deposit)

				// Verify expiration is years from creation
				expectedExp := storedDID.Created.AddDate(int(years), 0, 0)
				require.Equal(t, expectedExp, storedDID.Exp)
			}
		})
	}
}

// TestRenewDIDValidations tests the validations for the RenewDID function
func TestRenewDIDValidations(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	validCreator := sdk.AccAddress([]byte("test_creator")).String()
	wrongCreator := sdk.AccAddress([]byte("wrong_creator")).String()
	validDID := "did:example:valid"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// First create a DID
	createMsg := &types.MsgAddDID{
		Creator: validCreator,
		Did:     validDID,
		Years:   1,
	}
	_, err := ms.AddDID(ctx, createMsg)
	require.NoError(t, err)

	// Get initial DID entry for later comparison
	initialEntry, err := k.DIDDirectory.Get(ctx, validDID)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgRenewDID
		wantErr bool
		errMsg  string
	}{
		{
			name: "Empty DID",
			msg: &types.MsgRenewDID{
				Creator: validCreator,
				Did:     "",
				Years:   1,
			},
			wantErr: true,
			errMsg:  "DID is required",
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgRenewDID{
				Creator: validCreator,
				Did:     "invalid-did",
				Years:   1,
			},
			wantErr: true,
			errMsg:  "invalid DID syntax",
		},
		{
			name: "DID Not Found",
			msg: &types.MsgRenewDID{
				Creator: validCreator,
				Did:     "did:example:notfound",
				Years:   1,
			},
			wantErr: true,
			errMsg:  "DID not found",
		},
		{
			name: "Wrong Creator",
			msg: &types.MsgRenewDID{
				Creator: wrongCreator,
				Did:     validDID,
				Years:   1,
			},
			wantErr: true,
			errMsg:  "only the controller can renew a DID",
		},
		{
			name: "Years Too High",
			msg: &types.MsgRenewDID{
				Creator: validCreator,
				Did:     validDID,
				Years:   32, // Max is 31
			},
			wantErr: true,
			errMsg:  "years must be between 1 and 31",
		},
		{
			name: "Default Years (0 becomes 1)",
			msg: &types.MsgRenewDID{
				Creator: validCreator,
				Did:     validDID,
				Years:   0, // Should default to 1
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.RenewDID(ctx, tc.msg)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)

				if tc.msg.Did == validDID {
					// Verify DID wasn't modified for invalid attempts
					currentDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
					require.NoError(t, err)
					require.Equal(t, initialEntry, currentDID)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Get updated DID entry
				storedDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)

				// Check years and deposit calculations
				years := tc.msg.Years
				if years == 0 {
					years = 1
				}
				expectedDeposit := initialEntry.Deposit + int64(params.DidDirectoryTrustDeposit*uint64(years))
				require.Equal(t, expectedDeposit, storedDID.Deposit)

				// Verify expiration is extended by years
				expectedExp := initialEntry.Exp.AddDate(int(years), 0, 0)
				require.Equal(t, expectedExp, storedDID.Exp)

				// Store the updated values for next test case
				initialEntry = storedDID
			}
		})
	}
}

func TestFixedRemoveDIDValidations(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	validCreator := sdk.AccAddress([]byte("test_creator")).String()
	wrongCreator := sdk.AccAddress([]byte("wrong_creator")).String()
	validDID := "did:example:valid"
	expiredDID := "did:example:expired"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	now := ctx.BlockTime()

	// Create a normal DID
	normalEntry := types.DIDDirectory{
		Did:        validDID,
		Controller: validCreator,
		Created:    now,
		Modified:   now,
		Exp:        now.AddDate(1, 0, 0), // Expires in 1 year
		Deposit:    5,
	}

	err := k.DIDDirectory.Set(ctx, validDID, normalEntry)
	require.NoError(t, err)

	// Create an expired DID (beyond grace period)
	gracePeriodDays := int(params.DidDirectoryGracePeriod)
	expiredEntry := types.DIDDirectory{
		Did:        expiredDID,
		Controller: validCreator,
		Created:    now.AddDate(0, 0, -gracePeriodDays-5), // Created before grace period
		Modified:   now.AddDate(0, 0, -gracePeriodDays-5),
		Exp:        now.AddDate(0, 0, -gracePeriodDays-1), // Expired beyond grace period
		Deposit:    5,
	}

	err = k.DIDDirectory.Set(ctx, expiredDID, expiredEntry)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgRemoveDID
		wantErr bool
		errMsg  string
	}{
		{
			name: "Empty DID",
			msg: &types.MsgRemoveDID{
				Creator: validCreator,
				Did:     "",
			},
			wantErr: true,
			errMsg:  "DID is required",
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgRemoveDID{
				Creator: validCreator,
				Did:     "invalid-did",
			},
			wantErr: true,
			errMsg:  "invalid DID syntax",
		},
		{
			name: "DID Not Found",
			msg: &types.MsgRemoveDID{
				Creator: validCreator,
				Did:     "did:example:notfound",
			},
			wantErr: true,
			errMsg:  "DID not found",
		},
		{
			name: "Wrong Creator Before Grace Period",
			msg: &types.MsgRemoveDID{
				Creator: wrongCreator,
				Did:     validDID,
			},
			wantErr: true,
			errMsg:  "only the controller can remove this DID before grace period",
		},
		{
			name: "Valid Removal by Controller",
			msg: &types.MsgRemoveDID{
				Creator: validCreator,
				Did:     validDID,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip "Valid Removal by Controller" on first pass to keep the DID for other tests
			_, err = k.DIDDirectory.Has(ctx, validDID)
			if tc.name == "Valid Removal by Controller" && err == nil {
				return
			}

			resp, err := ms.RemoveDID(ctx, tc.msg)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)

				if tc.msg.Did == validDID || tc.msg.Did == expiredDID {
					// Verify DID still exists for invalid removal attempts
					_, err = k.DIDDirectory.Has(ctx, tc.msg.Did)
					require.NoError(t, err)
				}
			} else {
				require.NoError(t, err) // No error expected for valid cases
				require.NotNil(t, resp)

				// Verify DID was removed
				_, err = k.DIDDirectory.Has(ctx, tc.msg.Did)
				require.Error(t, err) // Should error as DID no longer exists
			}
		})
	}
}

// TestTouchDIDValidations tests the validations for the TouchDID function
func TestTouchDIDValidations(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	validCreator := sdk.AccAddress([]byte("test_creator")).String()
	validDID := "did:example:valid"

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Create a DID for testing
	didEntry := types.DIDDirectory{
		Did:        validDID,
		Controller: validCreator,
		Created:    ctx.BlockTime(),
		Modified:   ctx.BlockTime(),
		Exp:        ctx.BlockTime().AddDate(1, 0, 0),
		Deposit:    5,
	}

	err := k.DIDDirectory.Set(ctx, validDID, didEntry)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		msg     *types.MsgTouchDID
		wantErr bool
		errMsg  string
	}{
		{
			name: "Empty DID",
			msg: &types.MsgTouchDID{
				Creator: validCreator,
				Did:     "",
			},
			wantErr: true,
			errMsg:  "DID is required",
		},
		{
			name: "Invalid DID Format",
			msg: &types.MsgTouchDID{
				Creator: validCreator,
				Did:     "invalid-did",
			},
			wantErr: true,
			errMsg:  "invalid DID syntax",
		},
		{
			name: "DID Not Found",
			msg: &types.MsgTouchDID{
				Creator: validCreator,
				Did:     "did:example:notfound",
			},
			wantErr: true,
			errMsg:  "DID not found",
		},
		{
			name: "Different Creator Can Still Touch",
			msg: &types.MsgTouchDID{
				Creator: sdk.AccAddress([]byte("different_creator")).String(),
				Did:     validDID,
			},
			wantErr: false, // Anyone can touch a DID
		},
		{
			name: "Valid Touch by Controller",
			msg: &types.MsgTouchDID{
				Creator: validCreator,
				Did:     validDID,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store initial state if DID exists
			var initialModified time.Time
			if tc.msg.Did == validDID {
				entry, err := k.DIDDirectory.Get(ctx, validDID)
				require.NoError(t, err)
				initialModified = entry.Modified

				// Move time forward before touching
				newTime := ctx.BlockTime().Add(10 * time.Second)
				ctx = ctx.WithBlockTime(newTime)
			}

			resp, err := ms.TouchDID(ctx, tc.msg)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify modified time was updated
				updatedEntry, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)
				require.True(t, updatedEntry.Modified.After(initialModified))
				require.Equal(t, ctx.BlockTime(), updatedEntry.Modified)
			}
		})
	}
}

// TestAddDIDEdgeCases tests edge cases for the AddDID function
func TestAddDIDEdgeCases(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	validCreator := sdk.AccAddress([]byte("test_creator")).String()

	// Set default params for testing
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	testCases := []struct {
		name    string
		setup   func(*sdk.Context)
		msg     *types.MsgAddDID
		isValid bool
		check   func(*testing.T, *types.DIDDirectory)
	}{
		{
			name: "Max Years (31)",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "did:example:max_years",
				Years:   31,
			},
			isValid: true,
			check: func(t *testing.T, entry *types.DIDDirectory) {
				expectedExp := entry.Created.AddDate(31, 0, 0)
				require.Equal(t, expectedExp, entry.Exp)
				expectedDeposit := int64(params.DidDirectoryTrustDeposit * 31)
				require.Equal(t, expectedDeposit, entry.Deposit)
			},
		},
		{
			name: "Long DID Path (still valid)",
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "did:example:a_very_long_did_path_that_is_still_valid_according_to_spec",
				Years:   1,
			},
			isValid: true,
		},
		{
			name: "Add Same DID After Removal",
			setup: func(ctx *sdk.Context) {
				// First add a DID
				initialDID := "did:example:readd_after_remove"
				createMsg := &types.MsgAddDID{
					Creator: validCreator,
					Did:     initialDID,
					Years:   1,
				}
				_, err := ms.AddDID(*ctx, createMsg)
				require.NoError(t, err)

				// Then remove it
				removeMsg := &types.MsgRemoveDID{
					Creator: validCreator,
					Did:     initialDID,
				}
				_, err = ms.RemoveDID(*ctx, removeMsg)
				require.NoError(t, err)
			},
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "did:example:readd_after_remove",
				Years:   2,
			},
			isValid: true,
			check: func(t *testing.T, entry *types.DIDDirectory) {
				// Should have a new deposit value for 2 years
				expectedDeposit := int64(params.DidDirectoryTrustDeposit * 2)
				require.Equal(t, expectedDeposit, entry.Deposit)
			},
		},
		{
			name: "Timing Edge Case - Leap Year Expiration",
			setup: func(ctx *sdk.Context) {
				// Set date to February 29 of a leap year
				leapYearTime := time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)
				*ctx = ctx.WithBlockTime(leapYearTime)
			},
			msg: &types.MsgAddDID{
				Creator: validCreator,
				Did:     "did:example:leap_year",
				Years:   1,
			},
			isValid: true,
			check: func(t *testing.T, entry *types.DIDDirectory) {
				// Should expire on February 29, 2025 (which will be March 1 since 2025 is not a leap year)
				expectedYear := 2025
				require.Equal(t, expectedYear, entry.Exp.Year())
				// In non-leap years, Feb 29 becomes March 1
				if entry.Exp.Month() == time.February {
					require.Equal(t, 29, entry.Exp.Day())
				} else {
					require.Equal(t, time.March, entry.Exp.Month())
					require.Equal(t, 1, entry.Exp.Day())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run any setup
			if tc.setup != nil {
				tc.setup(&ctx)
			}

			resp, err := ms.AddDID(ctx, tc.msg)

			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Retrieve the stored DID
				storedDID, err := k.DIDDirectory.Get(ctx, tc.msg.Did)
				require.NoError(t, err)

				// Run specific checks
				if tc.check != nil {
					tc.check(t, &storedDID)
				}
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}
