package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/td/keeper"
	"github.com/verana-labs/verana/x/td/types"
)

// TestAuthzCheck5_TrustDeposit verifies AUTHZ-CHECK-5 on the MOD-TD delegable
// messages: an unregistered signing corporation aborts with
// ErrCorporationNotRegistered; a registered one passes the check.
func TestAuthzCheck5_TrustDeposit(t *testing.T) {
	k, ctx, _, coKeeper := keepertest.TrustdepositKeeperWithCorp(t)
	ms := keeper.NewMsgServerImpl(k)
	corp := sdk.AccAddress([]byte("unregistered_corp___")).String()
	operator := sdk.AccAddress([]byte("operator____________")).String()

	t.Run("ReclaimTrustDepositYield: unregistered corporation aborts", func(t *testing.T) {
		coKeeper.Unregistered[corp] = true
		_, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    operator,
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})

	t.Run("ReclaimTrustDepositYield: registered corporation passes AUTHZ-CHECK-5", func(t *testing.T) {
		delete(coKeeper.Unregistered, corp)
		_, err := ms.ReclaimTrustDepositYield(ctx, &types.MsgReclaimTrustDepositYield{
			Corporation: corp,
			Operator:    operator,
		})
		// Fails later (no trust deposit exists), but NOT with the registration error.
		require.Error(t, err)
		require.NotErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})

	t.Run("RepaySlashedTrustDeposit: unregistered corporation aborts", func(t *testing.T) {
		coKeeper.Unregistered[corp] = true
		_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Operator:    operator,
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})
}
