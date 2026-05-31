package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/di/keeper"
	"github.com/verana-labs/verana/x/di/types"
)

// TestAuthzCheck5_StoreDigest verifies AUTHZ-CHECK-5 on MOD-DI MsgStoreDigest:
// an unregistered signing authority aborts with ErrCorporationNotRegistered
// (the registered/permissive path is covered by TestStoreDigest_HappyPath).
func TestAuthzCheck5_StoreDigest(t *testing.T) {
	mock := &keepertest.MockDelegationKeeper{}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("unregistered_corp___")).String()
	operator := sdk.AccAddress([]byte("operator_address____")).String()

	f.corpKeeper.unregistered[authority] = true
	_, err := ms.StoreDigest(f.ctx, &types.MsgStoreDigest{
		Authority: authority,
		Operator:  operator,
		Digest:    "sha256-abc123def456",
	})
	require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
}
