package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/xr/keeper"
	"github.com/verana-labs/verana/x/xr/types"
)

// TestAuthzCheck5_UpdateExchangeRate verifies AUTHZ-CHECK-5 on MOD-XR
// MsgUpdateExchangeRate: an unregistered signing authority aborts with
// ErrCorporationNotRegistered (the registered path is covered by
// TestUpdateExchangeRate_HappyPath).
func TestAuthzCheck5_UpdateExchangeRate(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("unregistered_corp___")).String()
	operator := sdk.AccAddress([]byte("operator_address____")).String()

	f.corpKeeper.unregistered[authority] = true
	_, err := ms.UpdateExchangeRate(f.ctx, &types.MsgUpdateExchangeRate{
		Authority: authority,
		Operator:  operator,
		Id:        1,
		Rate:      "200",
	})
	require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
}
