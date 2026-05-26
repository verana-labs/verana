package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/types"
)

func TestGenesis_InitExport_RoundTrip(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})

	gs := &types.GenesisState{
		Params: types.DefaultParams(),
		Corporations: []types.Corporation{
			{Id: 1, PolicyAddress: "cosmos1aaa", Did: "did:example:1", Created: time.Unix(1, 0), Modified: time.Unix(1, 0), Language: "en", ActiveVersion: 1},
			{Id: 2, PolicyAddress: "cosmos1bbb", Did: "did:example:2", Created: time.Unix(2, 0), Modified: time.Unix(2, 0), Language: "fr", ActiveVersion: 1},
		},
	}
	require.NoError(t, k.InitGenesis(ctx, *gs))

	// Counter bumped to highest id.
	cnt, err := k.Counter.Get(ctx, "co")
	require.NoError(t, err)
	require.Equal(t, uint64(2), cnt)

	// Reverse indexes populated.
	id, err := k.CorporationByPolicyAddr.Get(ctx, "cosmos1aaa")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
	id, err = k.CorporationByDID.Get(ctx, "did:example:2")
	require.NoError(t, err)
	require.Equal(t, uint64(2), id)

	got := k.ExportGenesis(ctx)
	require.Len(t, got.Corporations, 2)
}
