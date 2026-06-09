package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/td/types"
)

// [MOD-TD-MSG-1 / spec commit 90679a9] Execution-order fix: when augend exceeds
// td.refunded, missing_augend_share MUST be computed as (augend - td.refunded) /
// share_value BEFORE td.refunded is zeroed. The buggy ordering would compute the
// share from the full augend (refunded already zeroed), over-crediting shares.
func TestAdjustTrustDeposit_ExecutionOrder_MissingShareBeforeZeroing(t *testing.T) {
	k, _, ctx, coKeeper := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	require.NoError(t, k.SetParams(ctx, defaultTestParams())) // share value = 1.0

	testAccString := sdk.AccAddress([]byte("exec_order_acct__01")).String()
	corpID := coKeeper.IDFor(testAccString)

	// deposit=1000, refunded=200, share=1000.
	require.NoError(t, k.TrustDeposit.Set(ctx, corpID, types.TrustDeposit{
		CorporationId: corpID,
		Share:         math.LegacyNewDec(1000),
		Deposit:       1000,
		Refunded:      200,
	}))

	// Increase by 500: refunded(200) cannot fully cover it, so
	// neededDeposit = 500 - 200 = 300, missing_augend_share = 300 / 1.0 = 300.
	require.NoError(t, k.AdjustTrustDeposit(sdkCtx, testAccString, 500, "exec_order"))

	td, err := k.TrustDeposit.Get(ctx, corpID)
	require.NoError(t, err)
	require.Equal(t, uint64(0), td.Refunded, "refunded drained")
	require.Equal(t, uint64(1300), td.Deposit, "deposit += neededDeposit (300)")
	// Correct ordering: share += 300. Buggy ordering would add 500 -> share 1500.
	require.True(t, td.Share.Equal(math.LegacyNewDec(1300)),
		"share must increase by (augend-refunded)/share_value=300, got %s", td.Share)
}

// [MOD-TD-MSG-5] Slash invariant: refunded MUST NOT exceed the post-slash deposit.
// A slash that drops deposit below the outstanding refunded amount clips refunded
// to the new deposit (the excess is forfeit).
func TestSlashTrustDeposit_ClipsRefundedToPostSlashDeposit(t *testing.T) {
	k, ms, ctx, coKeeper := setupMsgServer(t)

	require.NoError(t, k.SetParams(ctx, defaultTestParams())) // share value = 1.0

	testAccString := sdk.AccAddress([]byte("slash_clip_acct_01")).String()
	corpID := coKeeper.IDFor(testAccString)

	// deposit=1000, refunded=900.
	require.NoError(t, k.TrustDeposit.Set(ctx, corpID, types.TrustDeposit{
		CorporationId: corpID,
		Share:         math.LegacyNewDec(1000),
		Deposit:       1000,
		Refunded:      900,
	}))

	// Slash 300 -> post-slash deposit = 700, which is < refunded (900).
	_, err := ms.SlashTrustDeposit(ctx, &types.MsgSlashTrustDeposit{
		Authority:     govAuthority(),
		CorporationId: corpID,
		Deposit:       math.NewInt(300),
		Reason:        "invariant_test",
	})
	require.NoError(t, err)

	td, err := k.TrustDeposit.Get(ctx, corpID)
	require.NoError(t, err)
	require.Equal(t, uint64(700), td.Deposit)
	require.Equal(t, uint64(700), td.Refunded, "refunded clipped to post-slash deposit")
	require.Equal(t, uint64(300), td.SlashedDeposit)
}
