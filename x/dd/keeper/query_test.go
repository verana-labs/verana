package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/dd/types"
)

func TestListDIDs(t *testing.T) {
	keeper, ctx := keepertest.DiddirectoryKeeper(t)

	// Set up test data
	now := time.Now()
	ctx = ctx.WithBlockTime(now)

	// Create test DIDs with different attributes
	dids := []struct {
		did        string
		controller string
		modified   time.Time
		expired    bool
		overGrace  bool
	}{
		{
			did:        "did:example:1",
			controller: "cosmos1controller1",
			modified:   now.Add(-1 * time.Hour),
			expired:    false,
			overGrace:  false,
		},
		{
			did:        "did:example:2",
			controller: "cosmos1controller2",
			modified:   now.Add(-2 * time.Hour),
			expired:    true,
			overGrace:  false,
		},
		{
			did:        "did:example:3",
			controller: "cosmos1controller1",
			modified:   now.Add(-3 * time.Hour),
			expired:    true,
			overGrace:  true,
		},
	}

	params := types.DefaultParams()
	err := keeper.SetParams(ctx, params)
	require.NoError(t, err)

	// Store test DIDs
	for _, d := range dids {
		expTime := now.Add(24 * time.Hour) // Future expiry for non-expired
		if d.expired {
			if d.overGrace {
				expTime = now.AddDate(0, 0, -int(params.DidDirectoryGracePeriod)-1) // Past grace period
			} else {
				expTime = now.AddDate(0, 0, -1) // Just expired
			}
		}

		didEntry := types.DIDDirectory{
			Did:        d.did,
			Controller: d.controller,
			Created:    d.modified,
			Modified:   d.modified,
			Exp:        expTime,
			Deposit:    5,
		}
		err = keeper.DIDDirectory.Set(ctx, d.did, didEntry)
		require.NoError(t, err)
	}

	testCases := []struct {
		name      string
		req       *types.QueryListDIDsRequest
		expected  int
		expectErr bool
	}{
		{
			name: "List All DIDs",
			req: &types.QueryListDIDsRequest{
				ResponseMaxSize: 10,
			},
			expected: 3,
		},
		{
			name: "Filter by Controller",
			req: &types.QueryListDIDsRequest{
				Account:         "cosmos1controller1",
				ResponseMaxSize: 10,
			},
			expected: 2,
		},
		{
			name: "Filter by Changed Time",
			req: &types.QueryListDIDsRequest{
				Changed:         &now,
				ResponseMaxSize: 10,
			},
			expected: 0,
		},
		{
			name: "Filter Expired",
			req: &types.QueryListDIDsRequest{
				Expired:         true,
				OverGrace:       false,
				ResponseMaxSize: 10,
			},
			expected: 2, // Should include both expired DIDs
		},
		{
			name: "Filter Over Grace",
			req: &types.QueryListDIDsRequest{
				Expired:         true,
				OverGrace:       true,
				ResponseMaxSize: 10,
			},
			expected: 1, // Should include only the over-grace DID
		},
		{
			name: "Invalid Response Size",
			req: &types.QueryListDIDsRequest{
				ResponseMaxSize: 1025,
			},
			expectErr: true,
		},
		{
			name:     "List with Default Response Size",
			req:      &types.QueryListDIDsRequest{},
			expected: 3,
		},
		{
			name: "List with Small Response Size",
			req: &types.QueryListDIDsRequest{
				ResponseMaxSize: 2,
			},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := keeper.ListDIDs(ctx, tc.req)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, response)
			require.Len(t, response.Dids, tc.expected)

			// Verify sorting by modified time
			for i := 1; i < len(response.Dids); i++ {
				require.True(t, response.Dids[i-1].Modified.Before(response.Dids[i].Modified))
			}
		})
	}
}

func TestQueryGetDID(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	creator := sdk.AccAddress([]byte("test_creator")).String()
	validDID := "did:example:123456789abcdefghi"

	// Create a DID for testing
	createMsg := &types.MsgAddDID{
		Creator: creator,
		Did:     validDID,
		Years:   1,
	}
	_, err := ms.AddDID(ctx, createMsg)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		request       *types.QueryGetDIDRequest
		expectedError bool
		check         func(*testing.T, *types.QueryGetDIDResponse)
	}{
		{
			name: "Valid Query",
			request: &types.QueryGetDIDRequest{
				Did: validDID,
			},
			expectedError: false,
			check: func(t *testing.T, response *types.QueryGetDIDResponse) {
				require.NotNil(t, response)
				require.Equal(t, validDID, response.DidEntry.Did)
				require.Equal(t, creator, response.DidEntry.Controller)
				require.False(t, response.DidEntry.Created.IsZero())
				require.False(t, response.DidEntry.Modified.IsZero())
				require.False(t, response.DidEntry.Exp.IsZero())
			},
		},
		{
			name:          "Nil Request",
			request:       nil,
			expectedError: true,
		},
		{
			name: "Empty DID",
			request: &types.QueryGetDIDRequest{
				Did: "",
			},
			expectedError: true,
		},
		{
			name: "Invalid DID Format",
			request: &types.QueryGetDIDRequest{
				Did: "invalid-did",
			},
			expectedError: true,
		},
		{
			name: "Non-existent DID",
			request: &types.QueryGetDIDRequest{
				Did: "did:example:nonexistent",
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := k.GetDID(ctx, tc.request)

			if tc.expectedError {
				require.Error(t, err)
				require.Nil(t, response)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, response)

			if tc.check != nil {
				tc.check(t, response)
			}
		})
	}
}
