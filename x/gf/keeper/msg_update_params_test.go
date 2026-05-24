package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestMsgUpdateParams(t *testing.T) {
	govAuthority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	t.Run("happy path with gov authority", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: govAuthority,
			Params:    types.DefaultParams(),
		})
		require.NoError(t, err)
	})

	t.Run("rejects non-authority signer", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)

		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: testCorp, // not the gov module address
			Params:    types.DefaultParams(),
		})
		require.Error(t, err)
	})
}
