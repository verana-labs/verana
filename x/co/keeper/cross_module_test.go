package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/keeper"
)

func TestCoAsGFCorporationKeeper(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1corp"}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	adapter := keeper.NewCoAsGFCorporationKeeper(k)

	// ResolveByPolicyAddress hit.
	view, ok := adapter.ResolveByPolicyAddress(ctx, "cosmos1corp")
	require.True(t, ok)
	require.Equal(t, uint64(1), view.Id)
	require.Equal(t, uint32(1), view.ActiveVersion)

	// Miss.
	_, ok = adapter.ResolveByPolicyAddress(ctx, "cosmos1nobody")
	require.False(t, ok)

	// GetByID hit + miss.
	view, ok = adapter.GetByID(ctx, 1)
	require.True(t, ok)
	require.Equal(t, "cosmos1corp", view.PolicyAddress)
	_, ok = adapter.GetByID(ctx, 999)
	require.False(t, ok)

	// SetActiveVersion happy path.
	require.NoError(t, adapter.SetActiveVersion(ctx, 1, 5))
	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(5), co.ActiveVersion)

	// SetActiveVersion on missing corp returns error.
	require.Error(t, adapter.SetActiveVersion(ctx, 999, 5))
}
