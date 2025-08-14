package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/trustdeposit/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.TrustdepositKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
