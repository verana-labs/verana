package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

// TestCreateInitialGFVersionForCorporation_SetsActiveSince pins the spec
// requirement (MOD-CO-MSG-1-3) that the v1 GFV seeded for a new Corporation
// is born active with `active_since = block time`. Previously this was left
// as the zero time.Time{} and silently propagated to client responses.
func TestCreateInitialGFVersionForCorporation_SetsActiveSince(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	blockTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)

	err := k.CreateInitialGFVersionForCorporation(ctx, 42, "en", "https://x.example/c.pdf", "sha256-aGVsbG8=")
	require.NoError(t, err)

	// Direct lookup: the only GFV must be born active.
	count := 0
	require.NoError(t, k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		count++
		require.Equal(t, uint64(42), gfv.CorporationId)
		require.Zero(t, gfv.EcosystemId)
		require.Equal(t, uint32(1), gfv.Version)
		require.Equal(t, blockTime, gfv.Created)
		require.Equal(t, blockTime, gfv.ActiveSince, "spec MOD-CO-MSG-1-3: gfv.active_since MUST be set to current timestamp on seed")
		return false, nil
	}))
	require.Equal(t, 1, count)

	// ListVersionsByCorporation propagates the same value to the response shape.
	versions, err := k.ListVersionsByCorporation(ctx, 42, 1, false, "")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, blockTime, versions[0].ActiveSince, "ActiveSince must propagate through the query layer")
}

// TestCreateInitialGFVersionForCorporation_RejectsDoubleSeed pins the
// idempotency contract: calling the seeder twice for the same corp_id must
// abort cleanly. Spec mandates this is a single seed at Corporation creation;
// any second call is a bug in the caller.
func TestCreateInitialGFVersionForCorporation_RejectsDoubleSeed(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	require.NoError(t, k.CreateInitialGFVersionForCorporation(ctx, 1, "en", "https://x", "sha256-aGVsbG8="))
	err := k.CreateInitialGFVersionForCorporation(ctx, 1, "en", "https://x", "sha256-aGVsbG8=")
	require.ErrorIs(t, err, types.ErrInvalidVersion)
}

// TestCreateInitialGFVersionForCorporation_RejectsZeroCorpID pins the
// defensive check on corp_id = 0 (Cosmos collections allow uint64 zero keys,
// but spec mandates Corporation.id >= 1 so zero indicates a caller bug).
func TestCreateInitialGFVersionForCorporation_RejectsZeroCorpID(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	err := k.CreateInitialGFVersionForCorporation(ctx, 0, "en", "https://x", "sha256-aGVsbG8=")
	require.ErrorIs(t, err, types.ErrInvalidSubject)
}

// TestCreateInitialGFVersionForEcosystem_SetsActiveSince mirrors the
// Corporation test but exercises the ecosystem-owned XOR half of the GFV
// (EcosystemId set, CorporationId = 0). Pins spec MOD-ES-MSG-1-3: the v1
// GFV seeded for a new Ecosystem is born active with `active_since = block
// time`.
func TestCreateInitialGFVersionForEcosystem_SetsActiveSince(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	blockTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(blockTime)

	err := k.CreateInitialGFVersionForEcosystem(ctx, 7, "en", "https://x.example/e.pdf", "sha256-aGVsbG8=")
	require.NoError(t, err)

	count := 0
	require.NoError(t, k.GFVersion.Walk(ctx, nil, func(_ uint64, gfv types.GovernanceFrameworkVersion) (bool, error) {
		count++
		require.Equal(t, uint64(7), gfv.EcosystemId)
		require.Zero(t, gfv.CorporationId, "ecosystem-owned GFV MUST have CorporationId=0 (XOR invariant)")
		require.Equal(t, uint32(1), gfv.Version)
		require.Equal(t, blockTime, gfv.Created)
		require.Equal(t, blockTime, gfv.ActiveSince, "spec MOD-ES-MSG-1-3: gfv.active_since MUST be set to current timestamp on seed")
		return false, nil
	}))
	require.Equal(t, 1, count)

	// ListVersionsByEcosystem propagates the same value to the response shape.
	versions, err := k.ListVersionsByEcosystem(ctx, 7, 1, false, "")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, blockTime, versions[0].ActiveSince, "ActiveSince must propagate through the query layer")
	require.Equal(t, uint64(7), versions[0].EcosystemId)
}

// TestCreateInitialGFVersionForEcosystem_RejectsDoubleSeed pins idempotency:
// a second seed for the same ec_id must abort cleanly.
func TestCreateInitialGFVersionForEcosystem_RejectsDoubleSeed(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	require.NoError(t, k.CreateInitialGFVersionForEcosystem(ctx, 1, "en", "https://x", "sha256-aGVsbG8="))
	err := k.CreateInitialGFVersionForEcosystem(ctx, 1, "en", "https://x", "sha256-aGVsbG8=")
	require.ErrorIs(t, err, types.ErrInvalidVersion)
}

// TestCreateInitialGFVersionForEcosystem_RejectsZeroEcosystemID pins the
// defensive check on ec_id = 0.
func TestCreateInitialGFVersionForEcosystem_RejectsZeroEcosystemID(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	err := k.CreateInitialGFVersionForEcosystem(ctx, 0, "en", "https://x", "sha256-aGVsbG8=")
	require.ErrorIs(t, err, types.ErrInvalidSubject)
}
