package keeper_test

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/gf/types"
)

func TestKeeperGenesis_InitExportRoundTrip(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	t0 := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)

	original := types.GenesisState{
		Params: types.DefaultParams(),
		Versions: []types.GovernanceFrameworkVersion{
			{Id: 1, EcosystemId: 7, Version: 1, Created: t0},
			{Id: 2, EcosystemId: 7, Version: 2, Created: t0.Add(time.Hour)},
			{Id: 3, CorporationId: 9, Version: 1, Created: t0},
		},
		Documents: []types.GovernanceFrameworkDocument{
			{Id: 1, GfvId: 1, Language: "en", Url: "https://example.com/v1-en.html", DigestSri: "sha384-x", Created: t0},
			{Id: 2, GfvId: 1, Language: "fr", Url: "https://example.com/v1-fr.html", DigestSri: "sha384-x", Created: t0},
			{Id: 3, GfvId: 3, Language: "en", Url: "https://example.com/corp-v1-en.html", DigestSri: "sha384-x", Created: t0},
		},
	}

	require.NoError(t, k.InitGenesis(ctx, original))

	exported := k.ExportGenesis(ctx)
	require.NotNil(t, exported)

	require.Equal(t, original.Params, exported.Params)

	// Sort both sides by id for deterministic comparison.
	sortVersions := func(vs []types.GovernanceFrameworkVersion) { sort.Slice(vs, func(i, j int) bool { return vs[i].Id < vs[j].Id }) }
	sortDocs := func(ds []types.GovernanceFrameworkDocument) { sort.Slice(ds, func(i, j int) bool { return ds[i].Id < ds[j].Id }) }
	sortVersions(original.Versions)
	sortVersions(exported.Versions)
	sortDocs(original.Documents)
	sortDocs(exported.Documents)
	require.Equal(t, original.Versions, exported.Versions)
	require.Equal(t, original.Documents, exported.Documents)

	// Counter restoration: next ids must be > max(observed id).
	nextGFV, err := k.GetNextID(ctx, "gfv")
	require.NoError(t, err)
	require.Equal(t, uint64(4), nextGFV, "next gfv id must continue past the highest imported id (3)")

	nextGFD, err := k.GetNextID(ctx, "gfd")
	require.NoError(t, err)
	require.Equal(t, uint64(4), nextGFD, "next gfd id must continue past the highest imported id (3)")
}

func TestKeeperGenesis_EmptyDefault(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})

	require.NoError(t, k.InitGenesis(ctx, *types.DefaultGenesis()))

	exported := k.ExportGenesis(ctx)
	require.NotNil(t, exported)
	require.Equal(t, types.DefaultParams(), exported.Params)
	require.Empty(t, exported.Versions)
	require.Empty(t, exported.Documents)
}
