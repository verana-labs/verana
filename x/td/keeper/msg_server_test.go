package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/td/keeper"
	"github.com/verana-labs/verana/x/td/types"
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
				Authority: testAccString,
				Operator:  testAccString,
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
					Share:     math.LegacyNewDec(1000),
					Amount:    1000,
					Claimable: 0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDepositYield{
				Authority: testAccString,
				Operator:  testAccString,
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
					Share:     math.LegacyNewDec(1000),
					Amount:    1000, // 1000 shares at 1.5 value = 1500 tokens total value
					Claimable: 0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgReclaimTrustDepositYield{
				Authority: testAccString,
				Operator:  testAccString,
			},
			expErr: false,
			check: func(resp *types.MsgReclaimTrustDepositYieldResponse) {
				// Expected yield: 1000 shares * 1.5 value = 1500 total value - 1000 deposited = 500 yield
				require.Equal(t, uint64(500), resp.ClaimedAmount)

				// Verify trust deposit was updated correctly
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				// Shares reduced by 500/1.5 = 333.33...
				expectedShare := math.LegacyNewDec(1000).Sub(math.LegacyMustNewDecFromStr("333.333333333333333333"))
				require.True(t, td.Share.Equal(expectedShare), "expected %s, got %s", expectedShare.String(), td.Share.String())
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
					Share:     math.LegacyNewDec(1000),
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
					Share:     math.LegacyNewDec(1000),
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
					Share:     math.LegacyNewDec(1000),
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
				require.Equal(t, uint64(300), td.Claimable)                                                        // 500 - 200 = 300
				require.Equal(t, uint64(800), td.Amount)                                                           // 1000 - 200 = 800
				require.True(t, td.Share.Equal(math.LegacyNewDec(800)), "expected 800, got %s", td.Share.String()) // 1000 - 200 = 800 (1:1 ratio)
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
					Share:     math.LegacyNewDec(1000),
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
				require.Equal(t, uint64(300), td.Claimable)                                                          // 200 + 100 = 300
				require.Equal(t, uint64(1000), td.Amount)                                                            // Unchanged
				require.True(t, td.Share.Equal(math.LegacyNewDec(1000)), "expected 1000, got %s", td.Share.String()) // Unchanged
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
					Share:     math.LegacyNewDec(1000),
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
					Share:     math.LegacyNewDec(1000),
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
				require.Equal(t, uint64(250), td.Claimable)                                                          // 300 - 50 = 250
				require.Equal(t, uint64(1000), td.Amount)                                                            // Unchanged
				require.True(t, td.Share.Equal(math.LegacyNewDec(1000)), "expected 1000, got %s", td.Share.String()) // Unchanged
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
			share      math.LegacyDec
			shareValue math.LegacyDec
			expected   uint64
		}{
			{
				share:      math.LegacyNewDec(100),
				shareValue: math.LegacyNewDec(1),
				expected:   100,
			},
			{
				share:      math.LegacyNewDec(100),
				shareValue: math.LegacyMustNewDecFromStr("1.5"),
				expected:   150,
			},
			{
				share:      math.LegacyNewDec(0),
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
			expected   math.LegacyDec
		}{
			{
				amount:     100,
				shareValue: math.LegacyNewDec(1),
				expected:   math.LegacyNewDec(100),
			},
			{
				amount:     150,
				shareValue: math.LegacyMustNewDecFromStr("1.5"),
				expected:   math.LegacyNewDec(100), // 150/1.5 = 100
			},
			{
				amount:     0,
				shareValue: math.LegacyNewDec(1),
				expected:   math.LegacyNewDec(0),
			},
			{
				amount:     100,
				shareValue: math.LegacyNewDec(0),
				expected:   math.LegacyZeroDec(), // Division by zero prevention
			},
		}

		for _, tc := range testCases {
			result := k.AmountToShare(tc.amount, tc.shareValue)
			require.True(t, result.Equal(tc.expected), "expected %s, got %s", tc.expected.String(), result.String())
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

// govAuthority returns the governance module address string used as the keeper's authority.
func govAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

func defaultTestParams() types.Params {
	return types.Params{
		TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
		TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
		TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
		WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
		UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
	}
}

func setupMsgServerWithDelegation(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context, *keepertest.MockDelegationKeeper) {
	k, ctx, dk := keepertest.TrustdepositKeeperWithDelegation(t)
	return k, keeper.NewMsgServerImpl(k), ctx, dk
}

// ============================================================================
// TestMsgSlashTrustDeposit
// ============================================================================

func TestMsgSlashTrustDeposit(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	testAddr := sdk.AccAddress([]byte("slash_target_addr_1"))
	testAccString := testAddr.String()

	testCases := []struct {
		name      string
		setup     func()
		msg       *types.MsgSlashTrustDeposit
		expErr    bool
		expErrMsg string
		check     func()
	}{
		{
			name: "Invalid authority",
			msg: &types.MsgSlashTrustDeposit{
				Authority: "verana1invalidauthority",
				Account:   testAccString,
				Amount:    math.NewInt(100),
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "Zero amount",
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   testAccString,
				Amount:    math.NewInt(0),
			},
			expErr:    true,
			expErrMsg: "amount must be greater than 0",
		},
		{
			name: "Negative amount",
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   testAccString,
				Amount:    math.NewInt(-100),
			},
			expErr:    true,
			expErrMsg: "amount must be greater than 0",
		},
		{
			name: "Trust deposit not found",
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   sdk.AccAddress([]byte("nonexistent_addr")).String(),
				Amount:    math.NewInt(100),
			},
			expErr:    true,
			expErrMsg: "trust deposit not found",
		},
		{
			name: "Insufficient trust deposit",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(100),
					Amount:  100,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   testAccString,
				Amount:    math.NewInt(200),
			},
			expErr:    true,
			expErrMsg: "insufficient trust deposit",
		},
		{
			name: "Successful slash",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(1000),
					Amount:  1000,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   testAccString,
				Amount:    math.NewInt(300),
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(700), td.Amount)
				require.Equal(t, uint64(300), td.SlashedDeposit)
				require.Equal(t, uint64(1), td.SlashCount)
				require.NotNil(t, td.LastSlashed)
				// share reduced by 300/1.0 = 300
				expectedShare := math.LegacyNewDec(700)
				require.True(t, td.Share.Equal(expectedShare), "expected %s, got %s", expectedShare, td.Share)
			},
		},
		{
			name: "Multiple slashes accumulate",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(1000),
					Amount:         1000,
					SlashedDeposit: 200,
					SlashCount:     1,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   testAccString,
				Amount:    math.NewInt(100),
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(900), td.Amount)
				require.Equal(t, uint64(300), td.SlashedDeposit) // 200 + 100
				require.Equal(t, uint64(2), td.SlashCount)       // 1 + 1
			},
		},
		{
			name: "Slash exact deposit amount",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(500),
					Amount:  500,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgSlashTrustDeposit{
				Authority: govAuthority(),
				Account:   testAccString,
				Amount:    math.NewInt(500),
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(0), td.Amount)
				require.Equal(t, uint64(500), td.SlashedDeposit)
				require.True(t, td.Share.Equal(math.LegacyZeroDec()), "expected 0, got %s", td.Share)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			_, err := ms.SlashTrustDeposit(ctx, tc.msg)
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

// ============================================================================
// TestMsgRepaySlashedTrustDeposit
// ============================================================================

func TestMsgRepaySlashedTrustDeposit(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	testAddr := sdk.AccAddress([]byte("repay_target_addr_1"))
	testAccString := testAddr.String()

	testCases := []struct {
		name      string
		setup     func()
		msg       *types.MsgRepaySlashedTrustDeposit
		expErr    bool
		expErrMsg string
		check     func()
	}{
		{
			name: "Trust deposit not found",
			msg: &types.MsgRepaySlashedTrustDeposit{
				Authority: sdk.AccAddress([]byte("nonexistent_addr__")).String(),
				Operator:  testAccString,
				Amount:    100,
			},
			expErr:    true,
			expErrMsg: "trust deposit entry not found",
		},
		{
			name: "Amount not equal to outstanding slash",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(700),
					Amount:         700,
					SlashedDeposit: 300,
					RepaidDeposit:  0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgRepaySlashedTrustDeposit{
				Authority: testAccString,
				Operator:  testAccString,
				Amount:    200, // outstanding is 300
			},
			expErr:    true,
			expErrMsg: "amount must exactly equal outstanding slashed amount",
		},
		{
			name: "Successful repay",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(700),
					Amount:         700,
					SlashedDeposit: 300,
					RepaidDeposit:  0,
					SlashCount:     1,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgRepaySlashedTrustDeposit{
				Authority: testAccString,
				Operator:  testAccString,
				Amount:    300,
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(1000), td.Amount)        // 700 + 300
				require.Equal(t, uint64(300), td.RepaidDeposit)  // 0 + 300
				require.Equal(t, uint64(300), td.SlashedDeposit) // unchanged
				require.NotNil(t, td.LastRepaid)
				// share increased by 300/1.0 = 300
				expectedShare := math.LegacyNewDec(1000) // 700 + 300
				require.True(t, td.Share.Equal(expectedShare), "expected %s, got %s", expectedShare, td.Share)
			},
		},
		{
			name: "Partial repay after prior repay",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(800),
					Amount:         800,
					SlashedDeposit: 500,
					RepaidDeposit:  200, // already partially repaid
					SlashCount:     2,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgRepaySlashedTrustDeposit{
				Authority: testAccString,
				Operator:  testAccString,
				Amount:    300, // outstanding = 500 - 200 = 300
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(1100), td.Amount)       // 800 + 300
				require.Equal(t, uint64(500), td.RepaidDeposit) // 200 + 300
			},
		},
		{
			name: "AUTHZ succeeds with authorized operator",
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(900),
					Amount:         900,
					SlashedDeposit: 100,
					RepaidDeposit:  0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			msg: &types.MsgRepaySlashedTrustDeposit{
				Authority: testAccString,
				Operator:  sdk.AccAddress([]byte("different_operator")).String(),
				Amount:    100,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			_, err := ms.RepaySlashedTrustDeposit(ctx, tc.msg)
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

// ============================================================================
// TestMsgRepaySlashedTrustDepositAuthz
// ============================================================================

func TestMsgRepaySlashedTrustDepositAuthz(t *testing.T) {
	k, ms, ctx, dk := setupMsgServerWithDelegation(t)

	testAddr := sdk.AccAddress([]byte("authz_repay_addr__1"))
	testAccString := testAddr.String()
	operatorAddr := sdk.AccAddress([]byte("authz_operator_ad1")).String()

	t.Run("Authorization check fails", func(t *testing.T) {
		err := k.SetParams(ctx, defaultTestParams())
		require.NoError(t, err)
		td := types.TrustDeposit{
			Account:        testAccString,
			Share:          math.LegacyNewDec(700),
			Amount:         700,
			SlashedDeposit: 300,
			RepaidDeposit:  0,
		}
		err = k.TrustDeposit.Set(ctx, testAccString, td)
		require.NoError(t, err)

		dk.ErrToReturn = fmt.Errorf("mock: operator not authorized")
		_, err = ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Authority: testAccString,
			Operator:  operatorAddr,
			Amount:    300,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
	})

	t.Run("Authorization check succeeds", func(t *testing.T) {
		err := k.SetParams(ctx, defaultTestParams())
		require.NoError(t, err)
		td := types.TrustDeposit{
			Account:        testAccString,
			Share:          math.LegacyNewDec(700),
			Amount:         700,
			SlashedDeposit: 300,
			RepaidDeposit:  0,
		}
		err = k.TrustDeposit.Set(ctx, testAccString, td)
		require.NoError(t, err)

		dk.ErrToReturn = nil
		_, err = ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Authority: testAccString,
			Operator:  operatorAddr,
			Amount:    300,
		})
		require.NoError(t, err)
	})
}

// ============================================================================
// TestBurnEcosystemSlashedTrustDeposit
// ============================================================================

func TestBurnEcosystemSlashedTrustDeposit(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	testAddr := sdk.AccAddress([]byte("burn_eco_target_ad1"))
	testAccString := testAddr.String()

	testCases := []struct {
		name      string
		account   string
		amount    uint64
		setup     func()
		expErr    bool
		expErrMsg string
		check     func()
	}{
		{
			name:      "Empty account",
			account:   "",
			amount:    100,
			expErr:    true,
			expErrMsg: "account cannot be empty",
		},
		{
			name:      "Zero amount",
			account:   testAccString,
			amount:    0,
			expErr:    true,
			expErrMsg: "amount must be greater than 0",
		},
		{
			name:      "Trust deposit not found",
			account:   sdk.AccAddress([]byte("nonexistent_burn_a1")).String(),
			amount:    100,
			expErr:    true,
			expErrMsg: "trust deposit entry not found",
		},
		{
			name:    "Amount exceeds deposit",
			account: testAccString,
			amount:  200,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(100),
					Amount:  100,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr:    true,
			expErrMsg: "amount exceeds available deposit",
		},
		{
			name:    "Zero share value in params",
			account: testAccString,
			amount:  50,
			setup: func() {
				params := defaultTestParams()
				params.TrustDepositShareValue = math.LegacyMustNewDecFromStr("0")
				err := k.SetParams(ctx, params)
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(100),
					Amount:  100,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr:    true,
			expErrMsg: "trust deposit share value cannot be zero",
		},
		{
			name:    "Successful burn",
			account: testAccString,
			amount:  300,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(1000),
					Amount:  1000,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(700), td.Amount)
				expectedShare := math.LegacyNewDec(700) // 1000 - 300/1.0
				require.True(t, td.Share.Equal(expectedShare), "expected %s, got %s", expectedShare, td.Share)
			},
		},
		{
			name:    "Burn entire deposit",
			account: testAccString,
			amount:  500,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account: testAccString,
					Share:   math.LegacyNewDec(500),
					Amount:  500,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(0), td.Amount)
				require.True(t, td.Share.Equal(math.LegacyZeroDec()) || !td.Share.IsNegative(),
					"share should be zero or non-negative, got %s", td.Share)
			},
		},
		{
			name:    "Does NOT update SlashedDeposit or SlashCount",
			account: testAccString,
			amount:  100,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(1000),
					Amount:         1000,
					SlashedDeposit: 50,
					SlashCount:     2,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(50), td.SlashedDeposit, "SlashedDeposit should be unchanged")
				require.Equal(t, uint64(2), td.SlashCount, "SlashCount should be unchanged")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			err := k.BurnEcosystemSlashedTrustDeposit(sdkCtx, tc.account, tc.amount)
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

// ============================================================================
// TestAdjustTrustDepositOnBehalf
// ============================================================================

func TestAdjustTrustDepositOnBehalf(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	testAddr := sdk.AccAddress([]byte("onbehalf_target_ad1"))
	testAccString := testAddr.String()
	funder := sdk.AccAddress([]byte("funder_address_0001"))

	testCases := []struct {
		name      string
		account   string
		funder    sdk.AccAddress
		amount    int64
		setup     func()
		expErr    bool
		expErrMsg string
		check     func()
	}{
		{
			name:      "Negative amount",
			account:   testAccString,
			funder:    funder,
			amount:    -100,
			expErr:    true,
			expErrMsg: "amount must be positive",
		},
		{
			name:      "Zero amount",
			account:   testAccString,
			funder:    funder,
			amount:    0,
			expErr:    true,
			expErrMsg: "amount must be positive",
		},
		{
			name:      "Empty account",
			account:   "",
			funder:    funder,
			amount:    100,
			expErr:    true,
			expErrMsg: "account cannot be empty",
		},
		{
			name:    "New TD created on behalf",
			account: testAccString,
			funder:  funder,
			amount:  500,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				// Ensure no existing TD
				_ = k.TrustDeposit.Remove(ctx, testAccString)
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(500), td.Amount)
				require.Equal(t, uint64(0), td.Claimable)
				expectedShare := math.LegacyNewDec(500) // 500/1.0
				require.True(t, td.Share.Equal(expectedShare), "expected %s, got %s", expectedShare, td.Share)
			},
		},
		{
			name:    "Existing TD increased on behalf",
			account: testAccString,
			funder:  funder,
			amount:  200,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:   testAccString,
					Share:     math.LegacyNewDec(1000),
					Amount:    1000,
					Claimable: 100,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(1200), td.Amount)   // 1000 + 200
				require.Equal(t, uint64(100), td.Claimable) // unchanged
				expectedShare := math.LegacyNewDec(1200)    // 1000 + 200/1.0
				require.True(t, td.Share.Equal(expectedShare), "expected %s, got %s", expectedShare, td.Share)
			},
		},
		{
			name:    "Slashed and unrepaid TD blocked",
			account: testAccString,
			funder:  funder,
			amount:  100,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(700),
					Amount:         700,
					SlashedDeposit: 300,
					RepaidDeposit:  0,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr:    true,
			expErrMsg: "trust deposit has been slashed and not repaid",
		},
		{
			name:    "Slashed but fully repaid TD allowed",
			account: testAccString,
			funder:  funder,
			amount:  100,
			setup: func() {
				err := k.SetParams(ctx, defaultTestParams())
				require.NoError(t, err)
				td := types.TrustDeposit{
					Account:        testAccString,
					Share:          math.LegacyNewDec(1000),
					Amount:         1000,
					SlashedDeposit: 300,
					RepaidDeposit:  300,
				}
				err = k.TrustDeposit.Set(ctx, testAccString, td)
				require.NoError(t, err)
			},
			expErr: false,
			check: func() {
				td, err := k.TrustDeposit.Get(ctx, testAccString)
				require.NoError(t, err)
				require.Equal(t, uint64(1100), td.Amount) // 1000 + 100
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			err := k.AdjustTrustDepositOnBehalf(sdkCtx, tc.account, tc.funder, tc.amount)
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

// ============================================================================
// Additional edge cases for existing tests
// ============================================================================

func TestMsgReclaimTrustDepositYieldEdgeCases(t *testing.T) {
	t.Run("Slashed deposit guard blocks yield claim", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		testAddr := sdk.AccAddress([]byte("yield_slash_guard1"))
		testAccString := testAddr.String()

		params := defaultTestParams()
		params.TrustDepositShareValue = math.LegacyMustNewDecFromStr("1.5")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		td := types.TrustDeposit{
			Account:        testAccString,
			Share:          math.LegacyNewDec(1000),
			Amount:         1000,
			SlashedDeposit: 100,
			RepaidDeposit:  0,
		}
		err = k.TrustDeposit.Set(ctx, testAccString, td)
		require.NoError(t, err)

		_, err = ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Authority: testAccString,
			Operator:  testAccString,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "deposit has been slashed and not repaid")
	})

	t.Run("Slashed but repaid allows yield claim", func(t *testing.T) {
		k, ms, ctx := setupMsgServer(t)
		testAddr := sdk.AccAddress([]byte("yield_repaid_ok__1"))
		testAccString := testAddr.String()

		params := defaultTestParams()
		params.TrustDepositShareValue = math.LegacyMustNewDecFromStr("1.5")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		td := types.TrustDeposit{
			Account:        testAccString,
			Share:          math.LegacyNewDec(1000),
			Amount:         1000,
			SlashedDeposit: 100,
			RepaidDeposit:  100, // fully repaid
		}
		err = k.TrustDeposit.Set(ctx, testAccString, td)
		require.NoError(t, err)

		resp, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Authority: testAccString,
			Operator:  testAccString,
		})
		require.NoError(t, err)
		require.Equal(t, uint64(500), resp.ClaimedAmount) // 1000*1.5 - 1000 = 500
	})

	t.Run("AUTHZ-CHECK fails", func(t *testing.T) {
		k, ms, ctx, dk := setupMsgServerWithDelegation(t)
		testAddr := sdk.AccAddress([]byte("yield_authz_fail_1"))
		testAccString := testAddr.String()

		params := defaultTestParams()
		params.TrustDepositShareValue = math.LegacyMustNewDecFromStr("1.5")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		td := types.TrustDeposit{
			Account: testAccString,
			Share:   math.LegacyNewDec(1000),
			Amount:  1000,
		}
		err = k.TrustDeposit.Set(ctx, testAccString, td)
		require.NoError(t, err)

		dk.ErrToReturn = fmt.Errorf("mock: not authorized")
		_, err = ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Authority: testAccString,
			Operator:  sdk.AccAddress([]byte("bad_operator_addr1")).String(),
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "authorization check failed")
	})

	t.Run("AUTHZ-CHECK succeeds with different operator", func(t *testing.T) {
		k, ms, ctx, dk := setupMsgServerWithDelegation(t)
		testAddr := sdk.AccAddress([]byte("yield_authz_pass_1"))
		testAccString := testAddr.String()

		params := defaultTestParams()
		params.TrustDepositShareValue = math.LegacyMustNewDecFromStr("1.5")
		err := k.SetParams(ctx, params)
		require.NoError(t, err)

		td := types.TrustDeposit{
			Account: testAccString,
			Share:   math.LegacyNewDec(1000),
			Amount:  1000,
		}
		err = k.TrustDeposit.Set(ctx, testAccString, td)
		require.NoError(t, err)

		dk.ErrToReturn = nil
		resp, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Authority: testAccString,
			Operator:  sdk.AccAddress([]byte("good_operator_ad_1")).String(),
		})
		require.NoError(t, err)
		require.Equal(t, uint64(500), resp.ClaimedAmount)
	})
}

func TestAdjustTrustDepositSlashedGuard(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	testAddr := sdk.AccAddress([]byte("adjust_slash_guard"))
	testAccString := testAddr.String()

	err := k.SetParams(ctx, defaultTestParams())
	require.NoError(t, err)

	td := types.TrustDeposit{
		Account:        testAccString,
		Share:          math.LegacyNewDec(700),
		Amount:         700,
		SlashedDeposit: 300,
		RepaidDeposit:  0,
	}
	err = k.TrustDeposit.Set(ctx, testAccString, td)
	require.NoError(t, err)

	err = k.AdjustTrustDeposit(sdkCtx, testAccString, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "slashed and not fully repaid")
}
