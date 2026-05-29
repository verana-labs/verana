package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
)

// TestEcAsGFEcosystemKeeper_GetEcosystemView pins that the adapter returns
// the real CorporationId (vs the interim TR adapter's hard-coded 0).
func TestEcAsGFEcosystemKeeper_GetEcosystemView(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 42)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)

	resp, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	adapter := keeper.NewEcAsGFEcosystemKeeper(k)
	view, ok := adapter.GetEcosystemView(ctx, resp.EcosystemId)
	require.True(t, ok)
	require.Equal(t, uint64(42), view.CorporationID, "view must surface the REAL corporation id (no longer the interim 0)")
	require.Equal(t, "en", view.Language)
	require.Equal(t, uint32(1), view.ActiveVersion)
}

func TestEcAsGFEcosystemKeeper_GetEcosystemView_NotFound(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	adapter := keeper.NewEcAsGFEcosystemKeeper(k)
	_, ok := adapter.GetEcosystemView(ctx, 999)
	require.False(t, ok)
}

// TestEcAsGFEcosystemKeeper_SetEcosystemActiveVersion pins the MOD-GF MSG-2
// callback: bumps ec.ActiveVersion + ec.Modified.
func TestEcAsGFEcosystemKeeper_SetEcosystemActiveVersion(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	ctx = ctx.WithBlockTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	bumpTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(bumpTime)
	adapter := keeper.NewEcAsGFEcosystemKeeper(k)
	require.NoError(t, adapter.SetEcosystemActiveVersion(ctx, 1, 5))

	ec, err := k.Ecosystem.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, uint32(5), ec.ActiveVersion)
	require.Equal(t, bumpTime, ec.Modified)
}

func TestEcAsGFEcosystemKeeper_SetEcosystemActiveVersion_NotFound(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	adapter := keeper.NewEcAsGFEcosystemKeeper(k)
	err := adapter.SetEcosystemActiveVersion(ctx, 999, 5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// Interface satisfaction is checked at compile time by the gftypes.EcosystemKeeper
// alias inside cross_module.go; this test just exercises the construction path.
var _ = types.AttributeKeyEcosystemID
