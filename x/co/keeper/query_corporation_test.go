package keeper_test

import (
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/co/types"
	gftypes "github.com/verana-labs/verana/x/gf/types"
)

func TestQueryGetCorporation_Happy(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1corp"}
	gf := &mockGF{listResp: []gftypes.GovernanceFrameworkVersionWithDocs{{Id: 7, Version: 1}}}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	resp, err := qs.GetCorporation(ctx, &types.QueryGetCorporationRequest{
		CorporationId:     1,
		ActiveGfOnly:      true,
		PreferredLanguage: "en",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.Corporation.Id)
	require.Equal(t, "cosmos1corp", resp.Corporation.PolicyAddress)
	require.Len(t, resp.Corporation.Versions, 1)
	require.Equal(t, uint64(7), resp.Corporation.Versions[0].Id)

	// Active-only flag + preferred lang forwarded.
	require.True(t, gf.listArgs.activeOnly)
	require.Equal(t, "en", gf.listArgs.preferredLang)
	require.Equal(t, uint64(1), gf.listArgs.corpID)
}

func TestQueryGetCorporation_BadInputs(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.GetCorporation(ctx, nil)
	require.Error(t, err)
	_, err = qs.GetCorporation(ctx, &types.QueryGetCorporationRequest{CorporationId: 0})
	require.Error(t, err)
	_, err = qs.GetCorporation(ctx, &types.QueryGetCorporationRequest{CorporationId: 999})
	require.Error(t, err) // not found
}

func TestQueryListCorporations_OrderingFilteringPagination(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1c1"}
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	t0 := time.Unix(1000, 0)
	t1 := time.Unix(2000, 0)
	t2 := time.Unix(3000, 0)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: t0})
	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	grp.policy = "cosmos1c2"
	msg2 := validCreateMsg(t)
	msg2.Did = "did:example:2"
	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: t1})
	_, err = ms.CreateCorporation(ctx, msg2)
	require.NoError(t, err)

	grp.policy = "cosmos1c3"
	msg3 := validCreateMsg(t)
	msg3.Did = "did:example:3"
	ctx = ctx.WithBlockHeader(cmtproto.Header{Time: t2})
	_, err = ms.CreateCorporation(ctx, msg3)
	require.NoError(t, err)

	// No filter, default page size → returns all 3 ordered by modified desc.
	resp, err := qs.ListCorporations(ctx, &types.QueryListCorporationsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Corporations, 3)
	require.Equal(t, uint64(3), resp.Corporations[0].Id) // newest first
	require.Equal(t, uint64(2), resp.Corporations[1].Id)
	require.Equal(t, uint64(1), resp.Corporations[2].Id)

	// modified_after t1 → only co3 (must be strictly after).
	after := t1
	resp, err = qs.ListCorporations(ctx, &types.QueryListCorporationsRequest{ModifiedAfter: &after})
	require.NoError(t, err)
	require.Len(t, resp.Corporations, 1)
	require.Equal(t, uint64(3), resp.Corporations[0].Id)

	// pagination clamp.
	resp, err = qs.ListCorporations(ctx, &types.QueryListCorporationsRequest{ResponseMaxSize: 2})
	require.NoError(t, err)
	require.Len(t, resp.Corporations, 2)

	// excessive page size rejected.
	_, err = qs.ListCorporations(ctx, &types.QueryListCorporationsRequest{ResponseMaxSize: 9999})
	require.Error(t, err)

	// nil request rejected.
	_, err = qs.ListCorporations(ctx, nil)
	require.Error(t, err)
}
