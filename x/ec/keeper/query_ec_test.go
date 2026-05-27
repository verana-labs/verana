package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
)

func TestGetEcosystem_Happy(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	resp, err := qs.GetEcosystem(ctx, &types.QueryGetEcosystemRequest{Id: 1, ActiveGfOnly: true, PreferredLanguage: "en"})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.Ecosystem.Id)
	require.Equal(t, "did:example:ec1", resp.Ecosystem.Did)
	require.Equal(t, uint64(1), resp.Ecosystem.CorporationId)
	require.False(t, resp.Ecosystem.Archived)

	// Query layer must delegate to gfKeeper for nested versions.
	require.Equal(t, 1, gf.listCalls)
	require.Equal(t, uint64(1), gf.listArgs.ecID)
	require.Equal(t, uint32(1), gf.listArgs.activeVersion)
	require.True(t, gf.listArgs.activeOnly)
	require.Equal(t, "en", gf.listArgs.preferredLang)
}

func TestGetEcosystem_NotFound(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.GetEcosystem(ctx, &types.QueryGetEcosystemRequest{Id: 999})
	require.Error(t, err)
}

func TestListEcosystems_FiltersByCorporationID(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.register(tkCorpB, 2)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	// Corp 1 → 2 ecosystems.
	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	msg := validCreateMsg(t)
	msg.Did = "did:example:two"
	_, err = ms.CreateEcosystem(ctx, msg)
	require.NoError(t, err)
	// Corp 2 → 1 ecosystem.
	msgB := validCreateMsg(t)
	msgB.Corporation = tkCorpB
	msgB.Did = "did:example:three"
	_, err = ms.CreateEcosystem(ctx, msgB)
	require.NoError(t, err)

	resp, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{CorporationId: 1})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 2, "corp 1 owns 2 ecosystems")

	resp, err = qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{CorporationId: 2})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 1)
	require.Equal(t, "did:example:three", resp.Ecosystems[0].Did)
}

// TestListEcosystems_DefaultOrderIsIDAsc pins the GAP 6.A decision: when
// modified_after is unset, results are sorted by id ASC (deterministic,
// stable).
func TestListEcosystems_DefaultOrderIsIDAsc(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	msg := validCreateMsg(t)
	msg.Did = "did:example:two"
	_, err = ms.CreateEcosystem(ctx, msg)
	require.NoError(t, err)

	resp, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 2)
	require.Equal(t, uint64(1), resp.Ecosystems[0].Id)
	require.Equal(t, uint64(2), resp.Ecosystems[1].Id)
}

// TestListEcosystems_ModifiedAfterSortsDesc pins MOD-ES-MSG-2-3:
// "If modified_after is specified, order by modified desc".
func TestListEcosystems_ModifiedAfterSortsDesc(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	// Create ec 1 at t0, ec 2 at t0+1m.
	ctx = ctx.WithBlockTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(time.Date(2026, 6, 1, 0, 1, 0, 0, time.UTC))
	msg := validCreateMsg(t)
	msg.Did = "did:example:two"
	_, err = ms.CreateEcosystem(ctx, msg)
	require.NoError(t, err)

	modAfter := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	resp, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{ModifiedAfter: &modAfter})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 2)
	require.Equal(t, uint64(2), resp.Ecosystems[0].Id, "newest first (modified DESC)")
	require.Equal(t, uint64(1), resp.Ecosystems[1].Id)
}

func TestListEcosystems_ResponseMaxSizeClamp(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{ResponseMaxSize: 2000})
	require.Error(t, err, "response_max_size > 1024 must reject")
}
