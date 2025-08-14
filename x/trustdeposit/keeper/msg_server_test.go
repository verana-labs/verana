package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/trustdeposit/keeper"
	"github.com/verana-labs/verana/x/trustdeposit/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

func TestMsgReclaimTrustDepositYield(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	// Create test account
	testAddr := sdk.AccAddress([]byte("test_address"))
	testAccString := testAddr.String()

	// Test cases
	testCases := []struct {
		name      string
		setup     func() // Setup function to prepare the test state
		msg       *types.MsgReclaimTrustDepositYield
		expErr    bool
		expErrMsg string
		check     func(*types.MsgReclaimTrustDepositYieldResponse) // Function to check response
	}{
		{
			name: "Trust deposit not found",
			msg: &types.MsgReclaimTrustDepositYield{
				Creator: testAccString,
			},
			expErr:    true,
			expErrMsg: "trust deposit not found",
		},
		{
			name: "No claimable yield",
			setup: func() {
				// Set params with no yield (share value = 1.0)
				params := types.Params{
					TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
					TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
					TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
					WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
					UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
				}
				err := k.SetParams(ctx, params)
				require.NoError(t, err)

				// Create a trust deposit with no yield
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDepositYield{
				Creator: testAccString,
			},
			expErr:    true,
			expErrMsg: "no claimable yield",
		},
		{
			name: "Successful yield claim",
			setup: func() {
				// Set params with yield (share value = 1.5)
				params := types.Params{
					TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.5"),
					TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
					TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
					WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
					UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
				}
				err := k.SetParams(ctx, params)
				require.NoError(t, err)

				// Create a trust deposit with potential yield
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000, // 1000 shares at 1.5 value = 1500 tokens total value
					Claimable: 0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDepositYield{
				Creator: testAccString,
			},
			expErr: false,
			check: func(resp *types.MsgReclaimTrustDepositYieldResponse) {
				// Expected yield: 1000 shares * 1.5 value = 1500 total value - 1000 deposited = 500 yield
				require.Equal(t, uint64(500), resp.ClaimedAmount)

				// Verify trust deposit was updated correctly
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				// Shares reduced by 500/1.5 = 333.33 rounded to 333
				require.Equal(t, uint64(667), td.Share)   // 1000 - 333 = 667
				require.Equal(t, uint64(1000), td.Amount) // Original deposit unchanged
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}

			resp, err := ms.ReclaimTrustDepositYield(ctx, tc.msg)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				if tc.check != nil {
					tc.check(resp)
				}
			}
		})
	}
}

func TestMsgReclaimTrustDeposit(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	// Create test account
	testAddr := sdk.AccAddress([]byte("test_address"))
	testAccString := testAddr.String()

	// Test cases
	testCases := []struct {
		name      string
		setup     func() // Setup function to prepare the test state
		msg       *types.MsgReclaimTrustDeposit
		expErr    bool
		expErrMsg string
		check     func(*types.MsgReclaimTrustDepositResponse) // Function to check response
	}{
		{
			name: "Zero claimed amount",
			msg: &types.MsgReclaimTrustDeposit{
				Creator: testAccString,
				Claimed: 0,
			},
			expErr:    true,
			expErrMsg: "claimed amount must be greater than 0",
		},
		{
			name: "Trust deposit not found",
			msg: &types.MsgReclaimTrustDeposit{
				Creator: testAccString,
				Claimed: 100,
			},
			expErr:    true,
			expErrMsg: "trust deposit not found",
		},
		{
			name: "Claimed exceeds claimable",
			setup: func() {
				// Set default params
				params := types.DefaultParams()
				err := k.SetParams(ctx, params)
				require.NoError(t, err)

				// Create a trust deposit with limited claimable amount
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 500,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDeposit{
				Creator: testAccString,
				Claimed: 600, // More than the 500 claimable
			},
			expErr:    true,
			expErrMsg: "claimed amount exceeds claimable balance",
		},
		{
			name: "Insufficient required minimum deposit",
			setup: func() {
				// Set params with a lower share value to make required minimum deposit less than remaining amount
				params := types.Params{
					TrustDepositShareValue:      math.LegacyMustNewDecFromStr("0.8"), // Makes required min deposit < actual deposit
					TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
					TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
					WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
					UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
				}
				err := k.SetParams(ctx, params)
				require.NoError(t, err)

				// Create a trust deposit
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 500,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDeposit{
				Creator: testAccString,
				Claimed: 100,
			},
			expErr:    true,
			expErrMsg: "insufficient required minimum deposit",
		},
		{
			name: "Successful reclaim",
			setup: func() {
				// Set params with 1:1 share value for simplicity
				params := types.Params{
					TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
					TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"), // 60% burn rate
					TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
					WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
					UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
				}
				err := k.SetParams(ctx, params)
				require.NoError(t, err)

				// Create a trust deposit with claimable amount
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 500,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDeposit{
				Creator: testAccString,
				Claimed: 200,
			},
			expErr: false,
			check: func(resp *types.MsgReclaimTrustDepositResponse) {
				// With 60% burn rate: 200 * 0.6 = 120 burned, 80 claimed
				require.Equal(t, uint64(120), resp.BurnedAmount)
				require.Equal(t, uint64(80), resp.ClaimedAmount)

				// Verify trust deposit was updated correctly
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(300), td.Claimable) // 500 - 200 = 300
				require.Equal(t, uint64(800), td.Amount)    // 1000 - 200 = 800
				require.Equal(t, uint64(800), td.Share)     // 1000 - 200 = 800 (1:1 ratio)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}

			resp, err := ms.ReclaimTrustDeposit(ctx, tc.msg)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				if tc.check != nil {
					tc.check(resp)
				}
			}
		})
	}
}

func TestAdjustTrustDeposit(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Create test account
	testAddr := sdk.AccAddress([]byte("test_address"))
	testAccString := testAddr.String()

	// Test cases
	testCases := []struct {
		name      string
		account   string
		augend    int64
		setup     func() // Setup function to prepare the test state
		expErr    bool
		expErrMsg string
		check     func() // Function to check state after execution
	}{
		{
			name:      "Invalid account address",
			account:   "invalid_address",
			augend:    100,
			expErr:    true,
			expErrMsg: "invalid account address",
		},
		{
			name:      "Zero augend",
			account:   testAccString,
			augend:    0,
			expErr:    true,
			expErrMsg: "augend must be non-zero",
		},
		{
			name:      "Decrease non-existent trust deposit",
			account:   testAccString,
			augend:    -100,
			expErr:    true,
			expErrMsg: "cannot decrease non-existent trust deposit",
		},
		{
			name:    "Successful decrease",
			account: testAccString,
			augend:  -100,
			setup: func() {
				// Create a trust deposit
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 200,
				}
				err := k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				// Verify trust deposit was updated correctly
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(300), td.Claimable) // 200 + 100 = 300
				require.Equal(t, uint64(1000), td.Amount)   // Unchanged
				require.Equal(t, uint64(1000), td.Share)    // Unchanged
			},
		},
		{
			name:    "Decrease with claimable exceeding deposit",
			account: testAccString,
			augend:  -900,
			setup: func() {
				// Create a trust deposit
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 200,
				}
				err := k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr:    true,
			expErrMsg: "claimable after adjustment would exceed deposit",
		},
		{
			name:    "Increase using claimable",
			account: testAccString,
			augend:  50,
			setup: func() {
				// Create a trust deposit with claimable amount
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     1000,
					Amount:    1000,
					Claimable: 300,
				}
				err := k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				// Verify trust deposit was updated correctly
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(250), td.Claimable) // 300 - 50 = 250
				require.Equal(t, uint64(1000), td.Amount)   // Unchanged
				require.Equal(t, uint64(1000), td.Share)    // Unchanged
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}

			err := k.AdjustTrustDeposit(sdkCtx, tc.account, tc.augend)

			if tc.expErr {
				require.Error(t, err)
				if tc.expErrMsg != "" {
					require.Contains(t, err.Error(), tc.expErrMsg)
				}
			} else {
				require.NoError(t, err)

				if tc.check != nil {
					tc.check()
				}
			}
		})
	}
}

func TestUtilityFunctions(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Test ShareToAmount
	t.Run("ShareToAmount", func(t *testing.T) {
		testCases := []struct {
			share      uint64
			shareValue math.LegacyDec
			expected   uint64
		}{
			{
				share:      100,
				shareValue: math.LegacyNewDec(1),
				expected:   100,
			},
			{
				share:      100,
				shareValue: math.LegacyMustNewDecFromStr("1.5"),
				expected:   150,
			},
			{
				share:      0,
				shareValue: math.LegacyNewDec(1),
				expected:   0,
			},
		}

		for _, tc := range testCases {
			result := k.ShareToAmount(tc.share, tc.shareValue)
			require.Equal(t, tc.expected, result)
		}
	})

	// Test AmountToShare
	t.Run("AmountToShare", func(t *testing.T) {
		testCases := []struct {
			amount     uint64
			shareValue math.LegacyDec
			expected   uint64
		}{
			{
				amount:     100,
				shareValue: math.LegacyNewDec(1),
				expected:   100,
			},
			{
				amount:     150,
				shareValue: math.LegacyMustNewDecFromStr("1.5"),
				expected:   100, // 150/1.5 = 100
			},
			{
				amount:     0,
				shareValue: math.LegacyNewDec(1),
				expected:   0,
			},
			{
				amount:     100,
				shareValue: math.LegacyNewDec(0),
				expected:   0, // Division by zero prevention
			},
		}

		for _, tc := range testCases {
			result := k.AmountToShare(tc.amount, tc.shareValue)
			require.Equal(t, tc.expected, result)
		}
	})

	// Test CalculateBurnAmount
	t.Run("CalculateBurnAmount", func(t *testing.T) {
		testCases := []struct {
			claimed  uint64
			burnRate math.LegacyDec
			expected uint64
		}{
			{
				claimed:  1000,
				burnRate: math.LegacyMustNewDecFromStr("0.6"),
				expected: 600, // 1000 * 0.6 = 600
			},
			{
				claimed:  1000,
				burnRate: math.LegacyMustNewDecFromStr("0"),
				expected: 0,
			},
			{
				claimed:  0,
				burnRate: math.LegacyMustNewDecFromStr("0.6"),
				expected: 0,
			},
		}

		for _, tc := range testCases {
			result := k.CalculateBurnAmount(tc.claimed, tc.burnRate)
			require.Equal(t, tc.expected, result)
		}
	})

	// Test parameter getters
	t.Run("Parameter getters", func(t *testing.T) {
		// Set custom params for testing
		params := types.Params{
			TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
			UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.3"),
			WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.4"),
			TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.5"),
			TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
		}

		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		// Test each getter function
		rate := k.GetTrustDepositRate(sdkCtx)
		require.Equal(t, params.TrustDepositRate, rate)

		userRate := k.GetUserAgentRewardRate(sdkCtx)
		require.Equal(t, params.UserAgentRewardRate, userRate)

		walletRate := k.GetWalletUserAgentRewardRate(sdkCtx)
		require.Equal(t, params.WalletUserAgentRewardRate, walletRate)

		shareValue := k.GetTrustDepositShareValue(sdkCtx)
		require.Equal(t, params.TrustDepositShareValue, shareValue)
	})
}
