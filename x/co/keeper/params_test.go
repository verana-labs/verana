package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/types"
)

func TestGetParams_DefaultFallbackWhenUnset(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	// Wipe the seeded params so the fallback path is exercised.
	require.NoError(t, k.Params.Remove(ctx))
	require.Equal(t, types.DefaultParams(), k.GetParams(ctx))
}

func TestSetGetParams_RoundTrip(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	require.Equal(t, types.DefaultParams(), k.GetParams(ctx))
}
