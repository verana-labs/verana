package types_test

import (
	keepertest "github.com/verana-labs/verana-blockchain/testutil/keeper"
	"github.com/verana-labs/verana-blockchain/testutil/nullify"
	trustregistry "github.com/verana-labs/verana-blockchain/x/tr/module"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-blockchain/x/tr/types"
)

func TestGenesis(t *testing.T) {
	// Create sample data for genesis
	now := time.Now().UTC()

	// Create a basic genesis state
	trustRegistries := []types.TrustRegistry{
		{
			Id:            1,
			Did:           "did:example:123",
			Controller:    "cosmos1abcdefg",
			Created:       now,
			Modified:      now,
			Archived:      nil,
			Deposit:       10000000,
			Aka:           "https://example.com",
			ActiveVersion: 1,
			Language:      "en",
		},
		{
			Id:            2,
			Did:           "did:example:456",
			Controller:    "cosmos1hijklmn",
			Created:       now,
			Modified:      now,
			Archived:      nil,
			Deposit:       20000000,
			Aka:           "https://example2.com",
			ActiveVersion: 1,
			Language:      "fr",
		},
	}

	gfVersions := []types.GovernanceFrameworkVersion{
		{
			Id:          1,
			TrId:        1,
			Created:     now,
			Version:     1,
			ActiveSince: now,
		},
		{
			Id:          2,
			TrId:        2,
			Created:     now,
			Version:     1,
			ActiveSince: now,
		},
	}

	gfDocuments := []types.GovernanceFrameworkDocument{
		{
			Id:        1,
			GfvId:     1,
			Created:   now,
			Language:  "en",
			Url:       "https://example.com/doc1",
			DigestSri: "sha384-abcdef1234567890",
		},
		{
			Id:        2,
			GfvId:     2,
			Created:   now,
			Language:  "fr",
			Url:       "https://example2.com/doc1",
			DigestSri: "sha384-0987654321fedcba",
		},
	}

	counters := []types.Counter{
		{EntityType: "tr", Value: 2},
		{EntityType: "gfv", Value: 2},
		{EntityType: "gfd", Value: 2},
	}

	genesisState := types.GenesisState{
		Params:                       types.DefaultParams(),
		TrustRegistries:              trustRegistries,
		GovernanceFrameworkVersions:  gfVersions,
		GovernanceFrameworkDocuments: gfDocuments,
		Counters:                     counters,
	}

	// Initialize and export genesis
	k, ctx := keepertest.TrustregistryKeeper(t)
	trustregistry.InitGenesis(ctx, k, genesisState)
	got := trustregistry.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	// Check the exported genesis matches what we put in
	// We use nullify to ignore fields that are difficult to compare
	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// Compare params
	require.Equal(t, genesisState.Params, got.Params)

	// Compare trust registries
	require.ElementsMatch(t, genesisState.TrustRegistries, got.TrustRegistries)

	// Compare governance framework versions
	require.ElementsMatch(t, genesisState.GovernanceFrameworkVersions, got.GovernanceFrameworkVersions)

	// Compare governance framework documents
	require.ElementsMatch(t, genesisState.GovernanceFrameworkDocuments, got.GovernanceFrameworkDocuments)

	// Compare counters
	require.Equal(t, genesisState.Counters, got.Counters)

	// Verify the data can be retrieved from the keeper
	// Check trust registries
	for _, tr := range trustRegistries {
		gotTR, err := k.TrustRegistry.Get(ctx, tr.Id)
		require.NoError(t, err)
		require.Equal(t, nullify.Fill(tr), nullify.Fill(gotTR))

		// Check DID index
		trID, err := k.TrustRegistryDIDIndex.Get(ctx, tr.Did)
		require.NoError(t, err)
		require.Equal(t, tr.Id, trID)
	}

	// Check governance framework versions
	for _, gfv := range gfVersions {
		gotGFV, err := k.GFVersion.Get(ctx, gfv.Id)
		require.NoError(t, err)
		require.Equal(t, nullify.Fill(gfv), nullify.Fill(gotGFV))
	}

	// Check governance framework documents
	for _, gfd := range gfDocuments {
		gotGFD, err := k.GFDocument.Get(ctx, gfd.Id)
		require.NoError(t, err)
		require.Equal(t, nullify.Fill(gfd), nullify.Fill(gotGFD))
	}

	// Check counters
	for _, counter := range counters {
		gotValue, err := k.Counter.Get(ctx, counter.EntityType)
		require.NoError(t, err)
		require.Equal(t, counter.Value, gotValue)
	}
}

// Test additional scenarios
func TestInvalidGenesis(t *testing.T) {
	now := time.Now().UTC()

	// Test with duplicate trust registry IDs
	duplicateTRIDs := types.GenesisState{
		Params: types.DefaultParams(),
		TrustRegistries: []types.TrustRegistry{
			{
				Id:            1,
				Did:           "did:example:123",
				Controller:    "cosmos1abcdefg",
				Created:       now,
				Modified:      now,
				Language:      "en",
				ActiveVersion: 1,
			},
			{
				Id:            1, // Duplicate ID
				Did:           "did:example:456",
				Controller:    "cosmos1hijklmn",
				Created:       now,
				Modified:      now,
				Language:      "fr",
				ActiveVersion: 1,
			},
		},
	}

	err := duplicateTRIDs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate trust registry ID")

	// Test with duplicate DIDs
	duplicateDIDs := types.GenesisState{
		Params: types.DefaultParams(),
		TrustRegistries: []types.TrustRegistry{
			{
				Id:            1,
				Did:           "did:example:123",
				Controller:    "cosmos1abcdefg",
				Created:       now,
				Modified:      now,
				Language:      "en",
				ActiveVersion: 1,
			},
			{
				Id:            2,
				Did:           "did:example:123", // Duplicate DID
				Controller:    "cosmos1hijklmn",
				Created:       now,
				Modified:      now,
				Language:      "fr",
				ActiveVersion: 1,
			},
		},
	}

	err = duplicateDIDs.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate DID")

	// Test with invalid counter values
	invalidCounter := types.GenesisState{
		Params: types.DefaultParams(),
		TrustRegistries: []types.TrustRegistry{
			{
				Id:            5, // ID higher than counter
				Did:           "did:example:123",
				Controller:    "cosmos1abcdefg",
				Created:       now,
				Modified:      now,
				Language:      "en",
				ActiveVersion: 1,
			},
		},
		Counters: []types.Counter{
			{EntityType: "tr", Value: 3}, // Counter less than max ID
		},
	}

	err = invalidCounter.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "trust registry counter (3) is less than maximum trust registry ID (5)")

	// Test with duplicate counter entity types
	duplicateCounters := types.GenesisState{
		Params: types.DefaultParams(),
		Counters: []types.Counter{
			{EntityType: "tr", Value: 1},
			{EntityType: "gfv", Value: 1},
			{EntityType: "tr", Value: 2}, // Duplicate entity type
		},
	}

	err = duplicateCounters.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate counter entity type found")
}
