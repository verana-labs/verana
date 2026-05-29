package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
)

func TestArchiveEcosystem_Happy(t *testing.T) {
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

	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Archive:     true,
	})
	require.NoError(t, err)

	ec, err := k.Ecosystem.Get(ctx, 1)
	require.NoError(t, err)
	require.True(t, ec.Archived)
	require.Equal(t, bumpTime, ec.Modified)

	var found bool
	for _, e := range ctx.EventManager().Events() {
		if e.Type == types.EventTypeArchiveEcosystem {
			found = true
			attrs := map[string]string{}
			for _, a := range e.Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, "1", attrs[types.AttributeKeyEcosystemID])
			require.Equal(t, "archived", attrs[types.AttributeKeyArchiveStatus])
		}
	}
	require.True(t, found, "archive_ecosystem event must be emitted")
}

// TestArchiveEcosystem_UnarchiveHappy pins that archive=false on an archived
// row is the legitimate unarchive path.
func TestArchiveEcosystem_UnarchiveHappy(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{Corporation: tkCorp, Operator: tkOp, Id: 1, Archive: true})
	require.NoError(t, err)

	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{Corporation: tkCorp, Operator: tkOp, Id: 1, Archive: false})
	require.NoError(t, err)

	ec, err := k.Ecosystem.Get(ctx, 1)
	require.NoError(t, err)
	require.False(t, ec.Archived)
}

// TestArchiveEcosystem_IdempotencyAbortOnArchived pins MOD-ES-MSG-3-2-1:
// archive=true on an already-archived ecosystem must abort.
func TestArchiveEcosystem_IdempotencyAbortOnArchived(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{Corporation: tkCorp, Operator: tkOp, Id: 1, Archive: true})
	require.NoError(t, err)

	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{Corporation: tkCorp, Operator: tkOp, Id: 1, Archive: true})
	require.ErrorIs(t, err, types.ErrAlreadyInTargetArchiveState)
}

// TestArchiveEcosystem_IdempotencyAbortOnUnArchived pins the proto3-bool
// edge: submitting archive=false (or omitting it, which proto3 collapses to
// false) on an un-archived ecosystem must abort.
func TestArchiveEcosystem_IdempotencyAbortOnUnArchived(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{Corporation: tkCorp, Operator: tkOp, Id: 1, Archive: false})
	require.ErrorIs(t, err, types.ErrAlreadyInTargetArchiveState)
}

func TestArchiveEcosystem_AuthzDenied(t *testing.T) {
	del := &mockDelegation{err: errAuthDenied}
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, del, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Archive:     true,
	})
	require.ErrorIs(t, err, errAuthDenied)
}

func TestArchiveEcosystem_CorporationNotRegistered(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          1,
		Archive:     true,
	})
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
}

func TestArchiveEcosystem_EcosystemNotFound(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{
		Corporation: tkCorp,
		Operator:    tkOp,
		Id:          999,
		Archive:     true,
	})
	require.ErrorIs(t, err, types.ErrEcosystemNotFound)
}

func TestArchiveEcosystem_WrongCorporation(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.register(tkCorpB, 2)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)

	_, err = ms.ArchiveEcosystem(ctx, &types.MsgArchiveEcosystem{
		Corporation: tkCorpB, // wrong signer
		Operator:    tkOp,
		Id:          1,
		Archive:     true,
	})
	require.ErrorIs(t, err, types.ErrUnauthorizedOperator)
}
