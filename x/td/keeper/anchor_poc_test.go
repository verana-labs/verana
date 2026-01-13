package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/td/types"
)

// =============================================================================
// ANCHOR REGISTRATION TESTS
// =============================================================================

func TestAnchorRegistration(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create test anchor address (simulated group policy account)
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address1"))
	anchorID := anchorAddr.String()

	t.Run("Register anchor successfully", func(t *testing.T) {
		err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
		require.NoError(t, err)

		// Verify registration
		require.True(t, k.IsAnchor(sdkCtx, anchorID))

		// Verify anchor data
		anchor, err := k.GetAnchor(sdkCtx, anchorID)
		require.NoError(t, err)
		require.Equal(t, anchorID, anchor.AnchorId)
		require.Equal(t, uint64(1), anchor.GroupId)
		require.Equal(t, "Test Anchor", anchor.Metadata)
	})

	t.Run("Duplicate registration fails", func(t *testing.T) {
		err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Duplicate")
		require.Error(t, err)
		require.Contains(t, err.Error(), "already registered")
	})

	t.Run("Invalid address fails", func(t *testing.T) {
		err := k.RegisterAnchor(sdkCtx, "invalid_address", 1, "Invalid")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid anchor address")
	})

	t.Run("Non-existent anchor returns false", func(t *testing.T) {
		nonExistent := sdk.AccAddress([]byte("non_existent_anchor")).String()
		require.False(t, k.IsAnchor(sdkCtx, nonExistent))
	})
}

// =============================================================================
// VERIFIABLE SERVICE REGISTRATION TESTS
// =============================================================================

func TestVerifiableServiceRegistration(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup: Register anchor
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address2"))
	anchorID := anchorAddr.String()
	err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
	require.NoError(t, err)

	operatorAddr := sdk.AccAddress([]byte("test_operator_addr1"))
	operatorAccount := operatorAddr.String()

	t.Run("Register VS successfully", func(t *testing.T) {
		err := k.RegisterVerifiableService(sdkCtx, anchorID, operatorAccount, "VS Operator 1")
		require.NoError(t, err)

		// Verify registration
		require.True(t, k.IsVerifiableService(sdkCtx, operatorAccount))

		// Verify operator resolves to anchor
		resolvedAnchor, err := k.GetAnchorForOperator(sdkCtx, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, anchorID, resolvedAnchor)
	})

	t.Run("Re-register to same anchor updates metadata", func(t *testing.T) {
		err := k.RegisterVerifiableService(sdkCtx, anchorID, operatorAccount, "Updated Metadata")
		require.NoError(t, err)

		// Verify still resolves correctly
		resolvedAnchor, err := k.GetAnchorForOperator(sdkCtx, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, anchorID, resolvedAnchor)
	})

	t.Run("Register to different anchor fails", func(t *testing.T) {
		// Create second anchor
		anchor2Addr := sdk.AccAddress([]byte("test_anchor_address3"))
		anchor2ID := anchor2Addr.String()
		err := k.RegisterAnchor(sdkCtx, anchor2ID, 2, "Test Anchor 2")
		require.NoError(t, err)

		// Try to register same operator to different anchor
		err = k.RegisterVerifiableService(sdkCtx, anchor2ID, operatorAccount, "Should fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "already registered to anchor")
	})

	t.Run("Register to non-existent anchor fails", func(t *testing.T) {
		nonExistent := sdk.AccAddress([]byte("non_existent_anchor")).String()
		newOperator := sdk.AccAddress([]byte("new_operator_addr1")).String()
		err := k.RegisterVerifiableService(sdkCtx, nonExistent, newOperator, "Should fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "anchor not found")
	})

	t.Run("Invalid operator address fails", func(t *testing.T) {
		err := k.RegisterVerifiableService(sdkCtx, anchorID, "invalid_address", "Should fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid operator address")
	})
}

// =============================================================================
// OPERATOR ALLOWANCE TESTS
// =============================================================================

func TestOperatorAllowance(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup: Register anchor and VS
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address4"))
	anchorID := anchorAddr.String()
	err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
	require.NoError(t, err)

	operatorAddr := sdk.AccAddress([]byte("test_operator_addr2"))
	operatorAccount := operatorAddr.String()
	err = k.RegisterVerifiableService(sdkCtx, anchorID, operatorAccount, "VS Operator")
	require.NoError(t, err)

	t.Run("Set allowance successfully", func(t *testing.T) {
		err := k.SetOperatorAllowance(sdkCtx, anchorID, operatorAccount, 1000000, 86400)
		require.NoError(t, err)

		// Verify allowance
		allowance, err := k.GetOperatorAllowance(sdkCtx, anchorID, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, uint64(1000000), allowance.AllowanceLimit)
		require.Equal(t, uint64(0), allowance.Spent)
		require.Equal(t, uint64(86400), allowance.ResetPeriod)
	})

	t.Run("Set allowance for unregistered operator fails", func(t *testing.T) {
		unregistered := sdk.AccAddress([]byte("unregistered_op111")).String()
		err := k.SetOperatorAllowance(sdkCtx, anchorID, unregistered, 1000000, 86400)
		require.Error(t, err)
		require.Contains(t, err.Error(), "operator not registered")
	})

	t.Run("Set allowance for wrong anchor fails", func(t *testing.T) {
		wrongAnchor := sdk.AccAddress([]byte("wrong_anchor_addr1")).String()
		err := k.SetOperatorAllowance(sdkCtx, wrongAnchor, operatorAccount, 1000000, 86400)
		require.Error(t, err)
		require.Contains(t, err.Error(), "operator belongs to different anchor")
	})
}

// =============================================================================
// ANCHOR TRUST DEPOSIT ADJUSTMENT TESTS
// =============================================================================

func TestAnchorTrustDepositAdjustment(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup: Register anchor
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address5"))
	anchorID := anchorAddr.String()
	err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
	require.NoError(t, err)

	// Set params first
	params := types.Params{
		TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
		TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
		TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
		WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
		UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
	}
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	t.Run("Adjust trust deposit for non-existent anchor fails", func(t *testing.T) {
		nonExistent := sdk.AccAddress([]byte("non_existent_anchor")).String()
		err := k.AdjustAnchorTrustDeposit(sdkCtx, nonExistent, 1000000, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "anchor not found")
	})

	t.Run("Positive adjustment creates trust deposit", func(t *testing.T) {
		// This simulates a DID registration that increases trust deposit
		err := k.AdjustAnchorTrustDeposit(sdkCtx, anchorID, 5000000, "")
		require.NoError(t, err)

		// Verify
		savedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, anchorID, savedTd.Account)
		require.Equal(t, uint64(5000000), savedTd.Amount)
	})

	t.Run("Second positive adjustment accumulates", func(t *testing.T) {
		err := k.AdjustAnchorTrustDeposit(sdkCtx, anchorID, 1000000, "")
		require.NoError(t, err)

		savedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, uint64(6000000), savedTd.Amount) // 5M + 1M
	})

	t.Run("Negative adjustment without operator succeeds", func(t *testing.T) {
		// Direct anchor spending (no operator)
		err := k.AdjustAnchorTrustDeposit(sdkCtx, anchorID, -1000000, "")
		require.NoError(t, err)

		savedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, uint64(5000000), savedTd.Amount) // 6M - 1M
	})
}

// =============================================================================
// OPERATOR SPENDING TESTS
// =============================================================================

func TestOperatorSpending(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup: Anchor, VS, Allowance, Trust Deposit
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address6"))
	anchorID := anchorAddr.String()
	err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
	require.NoError(t, err)

	operatorAddr := sdk.AccAddress([]byte("test_operator_addr3"))
	operatorAccount := operatorAddr.String()
	err = k.RegisterVerifiableService(sdkCtx, anchorID, operatorAccount, "VS Operator")
	require.NoError(t, err)

	err = k.SetOperatorAllowance(sdkCtx, anchorID, operatorAccount, 1000000, 86400) // 1M limit per day
	require.NoError(t, err)

	// Set params and create trust deposit via adjustment
	params := types.Params{
		TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
		TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
		TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
		WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
		UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
	}
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create trust deposit via positive adjustment (simulates operations)
	err = k.AdjustAnchorTrustDeposit(sdkCtx, anchorID, 5000000, "")
	require.NoError(t, err)

	t.Run("Operator spends within limit", func(t *testing.T) {
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 500000, operatorAccount, "test_spend")
		require.NoError(t, err)

		// Verify debit
		updatedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, uint64(4500000), updatedTd.Amount) // 5M - 500K

		// Verify allowance updated
		allowance, err := k.GetOperatorAllowance(sdkCtx, anchorID, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, uint64(500000), allowance.Spent)
	})

	t.Run("Operator spends again within limit", func(t *testing.T) {
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 400000, operatorAccount, "test_spend_2")
		require.NoError(t, err)

		// Verify
		updatedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, uint64(4100000), updatedTd.Amount) // 4.5M - 400K

		allowance, err := k.GetOperatorAllowance(sdkCtx, anchorID, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, uint64(900000), allowance.Spent) // 500K + 400K
	})

	t.Run("Operator exceeds limit fails", func(t *testing.T) {
		// Remaining allowance is 100K, try to spend 200K
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 200000, operatorAccount, "should_fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds remaining allowance")

		// Verify nothing changed
		updatedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, uint64(4100000), updatedTd.Amount) // Unchanged
	})

	t.Run("Unauthorized operator fails", func(t *testing.T) {
		unauthorized := sdk.AccAddress([]byte("unauthorized_op0001")).String()
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 100000, unauthorized, "should_fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not authorized")
	})

	t.Run("Operator from different anchor fails", func(t *testing.T) {
		// Create different anchor with operator
		anchor2Addr := sdk.AccAddress([]byte("test_anchor_address7"))
		anchor2ID := anchor2Addr.String()
		err := k.RegisterAnchor(sdkCtx, anchor2ID, 2, "Test Anchor 2")
		require.NoError(t, err)

		op2Addr := sdk.AccAddress([]byte("test_operator_addr4"))
		op2Account := op2Addr.String()
		err = k.RegisterVerifiableService(sdkCtx, anchor2ID, op2Account, "VS Operator 2")
		require.NoError(t, err)

		// Try operator from anchor2 to spend from anchor1's TD
		err = k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 100000, op2Account, "should_fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "different anchor")
	})
}

// =============================================================================
// DIRECT ANCHOR SPENDING TESTS
// =============================================================================

func TestDirectAnchorSpending(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup: Anchor and Trust Deposit
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address8"))
	anchorID := anchorAddr.String()
	err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
	require.NoError(t, err)

	// Set params and create trust deposit via adjustment
	params := types.Params{
		TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
		TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
		TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
		WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
		UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
	}
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create trust deposit via positive adjustment
	err = k.AdjustAnchorTrustDeposit(sdkCtx, anchorID, 5000000, "")
	require.NoError(t, err)

	t.Run("Direct anchor spending (no operator) bypasses allowance check", func(t *testing.T) {
		// Empty operator = direct anchor spending, no limit check
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 2000000, "", "direct_anchor_spend")
		require.NoError(t, err)

		// Verify
		updatedTd, err := k.TrustDeposit.Get(ctx, anchorID)
		require.NoError(t, err)
		require.Equal(t, uint64(3000000), updatedTd.Amount) // 5M - 2M
	})

	t.Run("Direct spending fails on insufficient balance", func(t *testing.T) {
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 10000000, "", "should_fail")
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient trust deposit")
	})
}

// =============================================================================
// ALLOWANCE RESET TESTS
// =============================================================================

func TestAllowanceResetOnPeriodExpiry(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup
	anchorAddr := sdk.AccAddress([]byte("test_anchor_address9"))
	anchorID := anchorAddr.String()
	err := k.RegisterAnchor(sdkCtx, anchorID, 1, "Test Anchor")
	require.NoError(t, err)

	operatorAddr := sdk.AccAddress([]byte("test_operator_addr5"))
	operatorAccount := operatorAddr.String()
	err = k.RegisterVerifiableService(sdkCtx, anchorID, operatorAccount, "VS Operator")
	require.NoError(t, err)

	// Short reset period for testing (10 seconds)
	err = k.SetOperatorAllowance(sdkCtx, anchorID, operatorAccount, 1000000, 10)
	require.NoError(t, err)

	// Set params and create trust deposit via adjustment
	params := types.Params{
		TrustDepositShareValue:      math.LegacyMustNewDecFromStr("1.0"),
		TrustDepositRate:            math.LegacyMustNewDecFromStr("0.2"),
		TrustDepositReclaimBurnRate: math.LegacyMustNewDecFromStr("0.6"),
		WalletUserAgentRewardRate:   math.LegacyMustNewDecFromStr("0.3"),
		UserAgentRewardRate:         math.LegacyMustNewDecFromStr("0.2"),
	}
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create trust deposit via positive adjustment
	err = k.AdjustAnchorTrustDeposit(sdkCtx, anchorID, 5000000, "")
	require.NoError(t, err)

	t.Run("Spend and exhaust limit", func(t *testing.T) {
		err := k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 900000, operatorAccount, "test_spend")
		require.NoError(t, err)

		// Verify close to limit
		allowance, err := k.GetOperatorAllowance(sdkCtx, anchorID, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, uint64(900000), allowance.Spent)

		// Try to exceed limit
		err = k.DebitAnchorTrustDeposit(sdkCtx, anchorID, 200000, operatorAccount, "should_fail")
		require.Error(t, err)
	})

	t.Run("After period reset, allowance is replenished", func(t *testing.T) {
		// Create a new context with block time advanced by 15 seconds
		newBlockTime := sdkCtx.BlockTime().Add(15 * time.Second)
		newSdkCtx := sdkCtx.WithBlockTime(newBlockTime)

		// Now spending should work again
		err := k.DebitAnchorTrustDeposit(newSdkCtx, anchorID, 500000, operatorAccount, "after_reset")
		require.NoError(t, err)

		// Verify spent was reset to just this spend
		allowance, err := k.GetOperatorAllowance(newSdkCtx, anchorID, operatorAccount)
		require.NoError(t, err)
		require.Equal(t, uint64(500000), allowance.Spent)
	})
}
