package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana/x/co/types"
	"github.com/verana-labs/verana/x/de/keeper"
	"github.com/verana-labs/verana/x/de/types"
)

// TestAuthzCheck5_OperatorAuthorization verifies AUTHZ-CHECK-5 on the MOD-DE
// delegable Grant/Revoke messages: an unregistered signing corporation aborts
// with ErrCorporationNotRegistered (the registered path is covered by the
// existing Grant/Revoke happy-path tests via the permissive corp keeper).
func TestAuthzCheck5_OperatorAuthorization(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	corp := sdk.AccAddress([]byte("unregistered_corp___")).String()
	grantee := sdk.AccAddress([]byte("grantee_address_____")).String()
	f.corpKeeper.unregistered[corp] = true

	t.Run("GrantOperatorAuthorization: unregistered corporation aborts", func(t *testing.T) {
		_, err := ms.GrantOperatorAuthorization(f.ctx, &types.MsgGrantOperatorAuthorization{
			Corporation: corp,
			Operator:    "",
			Grantee:     grantee,
			MsgTypes:    []string{"/verana.ec.v1.MsgCreateEcosystem"},
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})

	t.Run("RevokeOperatorAuthorization: unregistered corporation aborts", func(t *testing.T) {
		_, err := ms.RevokeOperatorAuthorization(f.ctx, &types.MsgRevokeOperatorAuthorization{
			Corporation: corp,
			Operator:    "",
			Grantee:     grantee,
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})
}
