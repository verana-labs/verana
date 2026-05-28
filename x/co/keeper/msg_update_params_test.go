package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/co/types"
)

func TestMsgUpdateParams_AuthorityGating(t *testing.T) {
	t.Run("happy path: gov authority", func(t *testing.T) {
		k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			Params:    types.DefaultParams(),
		})
		require.NoError(t, err)
	})

	t.Run("rejects non-gov authority", func(t *testing.T) {
		k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
			Authority: "cosmos1somebodyelse",
			Params:    types.DefaultParams(),
		})
		require.Error(t, err)
	})

	t.Run("rejects empty authority", func(t *testing.T) {
		k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{Params: types.DefaultParams()})
		require.Error(t, err)
	})
}
