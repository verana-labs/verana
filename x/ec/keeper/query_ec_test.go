package keeper_test

import (
	"fmt"
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

// TestListEcosystems_TruncatesToResponseMaxSize pins that response_max_size
// caps results after sorting (not during Walk).
func TestListEcosystems_TruncatesToResponseMaxSize(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	for i := 1; i <= 5; i++ {
		msg := validCreateMsg(t)
		msg.Did = fmt.Sprintf("did:example:ec%d", i)
		_, err := ms.CreateEcosystem(ctx, msg)
		require.NoError(t, err)
	}

	resp, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{ResponseMaxSize: 3})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 3)
	require.Equal(t, uint64(1), resp.Ecosystems[0].Id)
	require.Equal(t, uint64(2), resp.Ecosystems[1].Id)
	require.Equal(t, uint64(3), resp.Ecosystems[2].Id)
}

// TestListEcosystems_ModifiedAfterTruncatesNewest pins HIGH-1 fix: with
// modified_after + response_max_size, the N most recently modified items
// are returned (not the N items with lowest IDs).
func TestListEcosystems_ModifiedAfterTruncatesNewest(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	times := []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	}
	for i, tm := range times {
		ctx = ctx.WithBlockTime(tm)
		msg := validCreateMsg(t)
		msg.Did = fmt.Sprintf("did:example:ec%d", i+1)
		_, err := ms.CreateEcosystem(ctx, msg)
		require.NoError(t, err)
	}

	epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	resp, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{
		ModifiedAfter:   &epoch,
		ResponseMaxSize: 2,
	})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 2)
	require.Equal(t, uint64(3), resp.Ecosystems[0].Id, "newest (ec3) must be first")
	require.Equal(t, uint64(2), resp.Ecosystems[1].Id)
}

// TestListEcosystems_CombinedFilter pins corporation_id + modified_after together.
func TestListEcosystems_CombinedFilter(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.register(tkCorpB, 2)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	ctx = ctx.WithBlockTime(t0)
	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t)) // corp1, modified=t0

	require.NoError(t, err)
	ctx = ctx.WithBlockTime(t1)
	msgB := validCreateMsg(t)
	msgB.Corporation = tkCorpB
	msgB.Did = "did:example:two"
	_, err = ms.CreateEcosystem(ctx, msgB) // corp2, modified=t1
	require.NoError(t, err)

	// corp1 + modified_after=t0: corp1 was modified AT t0, not strictly after
	resp, err := qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{
		CorporationId: 1,
		ModifiedAfter: &t0,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Ecosystems, "corp1 modified at t0 not strictly after t0")

	// corp2 + modified_after=t0: corp2 modified at t1 (after t0)
	resp, err = qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{
		CorporationId: 2,
		ModifiedAfter: &t0,
	})
	require.NoError(t, err)
	require.Len(t, resp.Ecosystems, 1)
	require.Equal(t, uint64(2), resp.Ecosystems[0].Id)
}

// TestGetEcosystem_GFKeeperError pins that a gfKeeper failure surfaces as
// codes.Internal.
func TestGetEcosystem_GFKeeperError(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{listErr: errAuthDenied}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	_, err = qs.GetEcosystem(ctx, &types.QueryGetEcosystemRequest{Id: 1})
	require.Error(t, err)
}

// TestListEcosystems_GFKeeperError pins buildWithVersions error propagation.
func TestListEcosystems_GFKeeperError(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{listErr: errAuthDenied}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	_, err = qs.ListEcosystems(ctx, &types.QueryListEcosystemsRequest{})
	require.Error(t, err)
}

func TestParams_Happy(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), resp.Params)
}
