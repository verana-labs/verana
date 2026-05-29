package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/ec/keeper"
	"github.com/verana-labs/verana/x/ec/types"
)

// Valid bech32 fixtures (deterministically generated from labels).
const (
	tkCorp  = "cosmos14wcc52lpsxwuxxhqjxrhvuumhm0xr6z247un93" // corp-test-1
	tkCorpB = "cosmos1lyfknrsmxhlr7rflvuz6x7jjjpnx4s5uywj78f" // corp-test-2
	tkOp    = "cosmos1fvz0kp4jfseea3zyduu78dd5yqcwrarwtxthjn" // operator-test
)

func validCreateMsg(t *testing.T) *types.MsgCreateEcosystem {
	t.Helper()
	return &types.MsgCreateEcosystem{
		Corporation:  tkCorp,
		Operator:     tkOp,
		Did:          "did:example:ec1",
		Language:     "en",
		DocUrl:       "https://example.com/ec.pdf",
		DocDigestSri: "sha256-aGVsbG8=",
	}
}

func TestCreateEcosystem_Happy(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	blockTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)

	resp, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.EcosystemId)

	// Ecosystem row persisted with correct shape.
	ec, err := k.Ecosystem.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), ec.CorporationId)
	require.Equal(t, "did:example:ec1", ec.Did)
	require.False(t, ec.Archived)
	require.Equal(t, uint32(1), ec.ActiveVersion)
	require.Equal(t, blockTime, ec.Created)
	require.Equal(t, blockTime, ec.Modified, "Modified MUST equal Created at creation")

	// GF seed was called with the right args (MOD-ES-MSG-1-3).
	require.Equal(t, 1, gf.createCalls)
	require.Equal(t, uint64(1), gf.createArgs.ecID)
	require.Equal(t, "en", gf.createArgs.language)
	require.Equal(t, "https://example.com/ec.pdf", gf.createArgs.docURL)

	// create_ecosystem event emitted with correct attributes.
	var found bool
	for _, e := range ctx.EventManager().Events() {
		if e.Type == types.EventTypeCreateEcosystem {
			found = true
			attrs := map[string]string{}
			for _, a := range e.Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, "1", attrs[types.AttributeKeyEcosystemID])
			require.Equal(t, "1", attrs[types.AttributeKeyCorporationID])
			require.Equal(t, "did:example:ec1", attrs[types.AttributeKeyDID])
			require.Equal(t, "en", attrs[types.AttributeKeyLanguage])
		}
	}
	require.True(t, found, "create_ecosystem event must be emitted")
}

func TestCreateEcosystem_ValidateBasicShortCircuits(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.CreateEcosystem(ctx, &types.MsgCreateEcosystem{}) // empty: ValidateBasic fails
	require.Error(t, err)
}

func TestCreateEcosystem_AuthzDenied(t *testing.T) {
	del := &mockDelegation{err: errAuthDenied}
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, del, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.ErrorIs(t, err, errAuthDenied)
	require.Equal(t, 1, del.calls)
}

// TestCreateEcosystem_CorporationNotRegistered pins AUTHZ-CHECK-5: signer must
// be the policy_address of an existing Corporation.
func TestCreateEcosystem_CorporationNotRegistered(t *testing.T) {
	k, ctx := ecKeeper(t, &mockDelegation{}, newMockCorporation(), &mockGF{})
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.ErrorIs(t, err, types.ErrCorporationNotRegistered)
}

// TestCreateEcosystem_DIDConsistency pins MOD-ES-MSG-1-2-1: existing
// Ecosystem with same did MUST share corporation_id (NOT unique-per-did).
func TestCreateEcosystem_DIDConsistency(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.register(tkCorpB, 2)
	gf := &mockGF{}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)

	// Same corp can create multiple ecosystems sharing one DID.
	resp1, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp1.EcosystemId)
	resp2, err := ms.CreateEcosystem(ctx, validCreateMsg(t)) // SAME did, SAME corp
	require.NoError(t, err, "same did across same-corp ecosystems is allowed")
	require.Equal(t, uint64(2), resp2.EcosystemId)

	// Different corp using the same DID must be rejected.
	msgB := validCreateMsg(t)
	msgB.Corporation = tkCorpB
	_, err = ms.CreateEcosystem(ctx, msgB)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

// TestCreateEcosystem_GFSeedFailureBubblesUp pins that gf failure propagates;
// in production the Cosmos SDK tx machinery rolls back the whole tx.
func TestCreateEcosystem_GFSeedFailureBubblesUp(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	gf := &mockGF{createErr: errAuthDenied}
	k, ctx := ecKeeper(t, &mockDelegation{}, co, gf)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "authorization denied")
}
