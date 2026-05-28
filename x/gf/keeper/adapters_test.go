package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	gfkeeper "github.com/verana-labs/verana/x/gf/keeper"
	gftypes "github.com/verana-labs/verana/x/gf/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"
)

// adapterCtx returns a fresh sdk.Context backed by a GF keeper test setup.
func adapterCtx(t *testing.T) sdk.Context {
	t.Helper()
	_, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	return ctx
}

// --- TRAsEcosystemKeeper -----------------------------------------------------

func TestTRAsEcosystemKeeper_GetEcosystemView_NotFound(t *testing.T) {
	trK, trCtx := keepertest.TrustregistryKeeper(t)
	adapter := gfkeeper.NewTRAsEcosystemKeeper(trK)

	// Fresh keeper has no entries.
	view, ok := adapter.GetEcosystemView(trCtx, 999)
	require.False(t, ok)
	require.Equal(t, gftypes.EcosystemView{}, view)
}

func TestTRAsEcosystemKeeper_GetEcosystemView_FoundReturnsZeroCorporationID(t *testing.T) {
	trK, trCtx := keepertest.TrustregistryKeeper(t)
	adapter := gfkeeper.NewTRAsEcosystemKeeper(trK)

	// Insert a TR directly into the underlying collection (bypass MsgCreate +
	// AUTHZ). The adapter is what we're testing; TR's create path has its own
	// tests.
	tr := trtypes.TrustRegistry{
		Id:            1,
		Did:           "did:example:1",
		Corporation:   "verana1corp00000000000000000000000000000abc",
		Language:      "en",
		ActiveVersion: 2,
		Created:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Modified:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, trK.TrustRegistry.Set(trCtx, tr.Id, tr))

	view, ok := adapter.GetEcosystemView(trCtx, 1)
	require.True(t, ok)
	require.Equal(t, uint64(1), view.Id)
	require.Equal(t, uint64(0), view.CorporationID, "interim adapter returns zero until MOD-CO resolver lands")
	require.Equal(t, "en", view.Language)
	require.Equal(t, uint32(2), view.ActiveVersion)
}

func TestTRAsEcosystemKeeper_SetEcosystemActiveVersion(t *testing.T) {
	trK, trCtx := keepertest.TrustregistryKeeper(t)
	adapter := gfkeeper.NewTRAsEcosystemKeeper(trK)

	original := trtypes.TrustRegistry{
		Id:            42,
		Did:           "did:example:42",
		Corporation:   "verana1corp00000000000000000000000000000abc",
		Language:      "en",
		ActiveVersion: 1,
		Created:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Modified:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, trK.TrustRegistry.Set(trCtx, original.Id, original))

	newBlockTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	trCtx = trCtx.WithBlockTime(newBlockTime)

	require.NoError(t, adapter.SetEcosystemActiveVersion(trCtx, 42, 5))

	got, err := trK.TrustRegistry.Get(trCtx, 42)
	require.NoError(t, err)
	require.Equal(t, uint32(5), got.ActiveVersion)
	require.Equal(t, newBlockTime, got.Modified, "adapter must bump tr.Modified to block time")
}

func TestTRAsEcosystemKeeper_SetEcosystemActiveVersion_NotFound(t *testing.T) {
	trK, trCtx := keepertest.TrustregistryKeeper(t)
	adapter := gfkeeper.NewTRAsEcosystemKeeper(trK)

	err := adapter.SetEcosystemActiveVersion(trCtx, 999, 1)
	require.Error(t, err, "must error when TR id doesn't exist")
}

// --- StubCorporationKeeper ---------------------------------------------------

func TestStubCorporationKeeper_AlwaysNotFound(t *testing.T) {
	stub := gfkeeper.NewStubCorporationKeeper()
	ctx := adapterCtx(t)

	resolved, ok := stub.ResolveByPolicyAddress(ctx, "any-policy-address")
	require.False(t, ok)
	require.Equal(t, gftypes.CorporationView{}, resolved)

	got, ok := stub.GetByID(ctx, 1)
	require.False(t, ok)
	require.Equal(t, gftypes.CorporationView{}, got)
}

func TestStubCorporationKeeper_SetActiveVersionAlwaysErrors(t *testing.T) {
	stub := gfkeeper.NewStubCorporationKeeper()
	ctx := adapterCtx(t)

	err := stub.SetActiveVersion(ctx, 1, 1)
	require.Error(t, err, "stub must error until MOD-CO lands")
	require.Contains(t, err.Error(), "corporation keeper not wired yet")
}

