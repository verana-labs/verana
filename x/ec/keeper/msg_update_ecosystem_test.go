package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
)

func TestUpdateEcosystem_Happy(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	ctx = ctx.WithBlockTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	bumpTime := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(bumpTime)

	_, err = ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Did:         "did:example:rotated",
	})
	require.NoError(t, err)

	ec, err := k.Ecosystem.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "did:example:rotated", ec.Did)
	require.Equal(t, bumpTime, ec.Modified)

	var found bool
	for _, e := range ctx.EventManager().Events() {
		if e.Type == types.EventTypeUpdateEcosystem {
			found = true
			attrs := map[string]string{}
			for _, a := range e.Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, "1", attrs[types.AttributeKeyEcosystemID])
			require.Equal(t, "did:example:rotated", attrs[types.AttributeKeyDID])
		}
	}
	require.True(t, found, "update_ecosystem event must be emitted")
}

// TestUpdateEcosystem_NoopOnSameDID pins the spec-deviation no-op branch
// (see msg_server.go method-level comment). AUTHZ-CHECK still runs but the
// persist+event path is skipped so `modified` is NOT bumped. Matches
// MOD-CO's UpdateCorporation precedent.
func TestUpdateEcosystem_NoopOnSameDID(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{}
	del := &mockDelegation{}
	k, ctx := ecKeeper(t, del, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	createTime := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(createTime)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	delCallsAfterCreate := del.calls

	ctx = ctx.WithBlockTime(time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC))
	_, err = ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Did:         "did:example:ec1", // unchanged
	})
	require.NoError(t, err)
	require.Equal(t, delCallsAfterCreate+1, del.calls, "AUTHZ-CHECK MUST run even on no-op")

	ec, err := k.Ecosystem.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, createTime, ec.Modified, "Modified MUST NOT bump on no-op did")
	for _, e := range ctx.EventManager().Events() {
		require.NotEqual(t, types.EventTypeUpdateEcosystem, e.Type, "no-op must not emit update event")
	}
}

func TestUpdateEcosystem_EcosystemNotFound(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          999,
		Did:         "did:example:any",
	})
	require.ErrorIs(t, err, types.ErrEcosystemNotFound)
}

// TestUpdateEcosystem_WrongCorporation pins that the signing Corporation MUST
// equal ecosystem.corporation_id.
func TestUpdateEcosystem_WrongCorporation(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.register(tkCorpB, 2)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t)) // owned by corp 1
	require.NoError(t, err)

	_, err = ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorpB, // corp 2 signs
		Operator:    tkOp,
		Id:          1,
		Did:         "did:example:rotated",
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedOperator)
}

func TestUpdateEcosystem_AuthzDenied(t *testing.T) {
	del := &mockDelegation{err: errAuthDenied}
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, del, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Did:         "did:example:any",
	})
	require.ErrorIs(t, err, errAuthDenied)
}

func TestUpdateEcosystem_CorporationNotRegistered(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Did:         "did:example:any",
	})
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
}

// TestUpdateEcosystem_DIDConflictWithOtherCorp pins MOD-ES-MSG-2-2-1:
// rotating to a did already owned by a different corp must abort.
func TestUpdateEcosystem_DIDConflictWithOtherCorp(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.register(tkCorpB, 2)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)

	// Corp 1 creates ec id=1 with did A.
	msgA := validCreateMsg(t)
	msgA.Did = "did:example:A"
	_, err := ms.CreateEcosystem(ctx, msgA)
	require.NoError(t, err)

	// Corp 2 creates ec id=2 with did B.
	msgB := validCreateMsg(t)
	msgB.Corporation = tkCorpB
	msgB.Did = "did:example:B"
	_, err = ms.CreateEcosystem(ctx, msgB)
	require.NoError(t, err)

	// Corp 1 tries to rotate ec 1 → did B (owned by corp 2).
	_, err = ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Did:         "did:example:B",
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}
