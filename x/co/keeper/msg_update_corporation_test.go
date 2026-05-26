package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/co/keeper"
	"github.com/verana-labs/verana/x/co/types"
)

func TestUpdateCorporation_Happy(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1corp"}
	gf := &mockGF{}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	_, err = ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: "cosmos1corp",
		Operator:    "cosmos1op",
		Did:         "did:example:rotated",
	})
	require.NoError(t, err)

	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "did:example:rotated", co.Did)

	// Index swap: old gone, new bound.
	_, err = k.CorporationByDID.Get(ctx, "did:example:1")
	require.Error(t, err)
	id, err := k.CorporationByDID.Get(ctx, "did:example:rotated")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
}

func TestUpdateCorporation_UnregisteredAccount(t *testing.T) {
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, &mockGroup{}, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: "cosmos1nobody",
		Operator:    "cosmos1op",
		Did:         "did:example:rotated",
	})
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
}

func TestUpdateCorporation_DEDenies(t *testing.T) {
	del := &mockDelegation{err: errAuthDenied}
	grp := &mockGroup{policy: "cosmos1corp"}
	k, ctx := keepertest.CoKeeper(t, del, grp, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err) // create doesn't go through DE

	_, err = ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: "cosmos1corp",
		Operator:    "cosmos1op",
		Did:         "did:example:rotated",
	})
	require.ErrorIs(t, err, errAuthDenied)
}

func TestUpdateCorporation_NoopOnSameDID(t *testing.T) {
	grp := &mockGroup{policy: "cosmos1corp"}
	del := &mockDelegation{}
	k, ctx := keepertest.CoKeeper(t, del, grp, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	_, err = ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: "cosmos1corp",
		Operator:    "cosmos1op",
		Did:         "did:example:1", // unchanged
	})
	require.NoError(t, err)
	// AUTHZ-CHECK-1 runs even on no-op: an unauthorized operator must not be
	// able to even attempt a write, regardless of whether the body mutates.
	require.Equal(t, 1, del.calls)
	// Modified must NOT advance and no update event emitted.
	co, err := k.Corporation.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "did:example:1", co.Did)
	for _, e := range ctx.EventManager().Events() {
		require.NotEqual(t, types.EventTypeUpdateCorporation, e.Type, "no-op must not emit update event")
	}
}

func TestUpdateCorporation_RejectsDIDAlreadyBoundElsewhere(t *testing.T) {
	gf := &mockGF{}
	grp := &mockGroup{policy: "cosmos1corp_a"}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t))
	require.NoError(t, err)

	grp.policy = "cosmos1corp_b"
	msg2 := validCreateMsg(t)
	msg2.Did = "did:example:2"
	_, err = ms.CreateCorporation(ctx, msg2)
	require.NoError(t, err)

	// Try to rotate corp_a → corp_b's DID.
	_, err = ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: "cosmos1corp_a",
		Operator:    "cosmos1op",
		Did:         "did:example:2",
	})
	require.ErrorIs(t, err, types.ErrDIDAlreadyExists)
}
