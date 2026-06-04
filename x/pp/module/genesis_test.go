package participant_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/testutil/nullify"
	participant "github.com/verana-labs/verana/x/pp/module"
	"github.com/verana-labs/verana/x/pp/types"
)

func TestGenesis(t *testing.T) {
	// Test default genesis state
	genesisState := types.GenesisState{
		Params:              types.DefaultParams(),
		Participants:        []types.Participant{},
		ParticipantSessions: []types.ParticipantSession{},
		NextParticipantId:   1,
	}

	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)
	participant.InitGenesis(ctx, k, genesisState)
	got := participant.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.Params, got.Params)
	require.ElementsMatch(t, genesisState.Participants, got.Participants)
	require.ElementsMatch(t, genesisState.ParticipantSessions, got.ParticipantSessions)
	require.Equal(t, genesisState.NextParticipantId, got.NextParticipantId)
}

func TestDeterministicGenesis(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)

	nowTime := time.Now()
	futureTime := nowTime.Add(24 * time.Hour)
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	// Create test participants in random order
	participant2 := types.Participant{
		Id:                     2,
		Role:                   types.ParticipantRole_ISSUER,
		Did:                    "did:example:67890",
		CorporationId:          uint64(1),
		Created:                &nowTime,
		Modified:               &nowTime,
		SchemaId:               1,
		ValidatorParticipantId: 1,
		EffectiveFrom:          &nowTime,
		EffectiveUntil:         &futureTime,
	}

	participant1 := types.Participant{
		Id:             1,
		Role:           types.ParticipantRole_ECOSYSTEM,
		Did:            "did:example:12345",
		CorporationId:  uint64(1),
		Created:        &nowTime,
		Modified:       &nowTime,
		SchemaId:       1,
		EffectiveFrom:  &nowTime,
		EffectiveUntil: &futureTime,
	}

	// Insert in reverse order
	require.NoError(t, k.Participant.Set(ctx, participant2.Id, participant2))
	require.NoError(t, k.Participant.Set(ctx, participant1.Id, participant1))
	require.NoError(t, k.ParticipantCounter.Set(ctx, 3))

	// Create test sessions in random order
	session2 := types.ParticipantSession{
		Id:            "test-session-id-2",
		CorporationId: uint64(1),
		VsOperator:    creatorAddr,
		Created:       &nowTime,
		Modified:      &nowTime,
		SessionRecords: []*types.ParticipantSessionRecord{
			{
				IssuerParticipantId: 1,
				Created:             &nowTime,
			},
		},
	}

	session1 := types.ParticipantSession{
		Id:            "test-session-id-1",
		CorporationId: uint64(1),
		VsOperator:    creatorAddr,
		Created:       &nowTime,
		Modified:      &nowTime,
		SessionRecords: []*types.ParticipantSessionRecord{
			{
				IssuerParticipantId: 1,
				Created:             &nowTime,
			},
		},
	}

	// Insert sessions in reverse order
	require.NoError(t, k.ParticipantSession.Set(ctx, session2.Id, session2))
	require.NoError(t, k.ParticipantSession.Set(ctx, session1.Id, session1))

	// Export genesis
	exportedGenesis1 := participant.ExportGenesis(ctx, k)

	// First export should have deterministic ordering
	require.Len(t, exportedGenesis1.Participants, 2)
	require.Len(t, exportedGenesis1.ParticipantSessions, 2)

	// Check if participants are sorted by ID
	require.Equal(t, uint64(1), exportedGenesis1.Participants[0].Id)
	require.Equal(t, uint64(2), exportedGenesis1.Participants[1].Id)

	// Check if sessions are sorted by ID
	require.Equal(t, "test-session-id-1", exportedGenesis1.ParticipantSessions[0].Id)
	require.Equal(t, "test-session-id-2", exportedGenesis1.ParticipantSessions[1].Id)

	// Create a new keeper instance for the second test
	k2, _, _, _, ctx2, _ := keepertest.ParticipantKeeper(t)

	// Insert in opposite order for the second test
	require.NoError(t, k2.Participant.Set(ctx2, participant1.Id, participant1))
	require.NoError(t, k2.Participant.Set(ctx2, participant2.Id, participant2))
	require.NoError(t, k2.ParticipantCounter.Set(ctx2, 3))

	// Insert sessions in opposite order
	require.NoError(t, k2.ParticipantSession.Set(ctx2, session1.Id, session1))
	require.NoError(t, k2.ParticipantSession.Set(ctx2, session2.Id, session2))

	// Export genesis again
	exportedGenesis2 := participant.ExportGenesis(ctx2, k2)

	// Second export should have same deterministic ordering despite different insertion order
	require.Len(t, exportedGenesis2.Participants, 2)
	require.Len(t, exportedGenesis2.ParticipantSessions, 2)

	// Check if participants are sorted by ID
	require.Equal(t, uint64(1), exportedGenesis2.Participants[0].Id)
	require.Equal(t, uint64(2), exportedGenesis2.Participants[1].Id)

	// Check if sessions are sorted by ID
	require.Equal(t, "test-session-id-1", exportedGenesis2.ParticipantSessions[0].Id)
	require.Equal(t, "test-session-id-2", exportedGenesis2.ParticipantSessions[1].Id)

	// The two exports should be identical despite different insertion orders
	nullify.Fill(exportedGenesis1)
	nullify.Fill(exportedGenesis2)
	require.Equal(t, exportedGenesis1, exportedGenesis2)
}

func TestGenesisImportExport(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)

	// Create some test data
	nowTime := time.Now()
	futureTime := nowTime.Add(24 * time.Hour)
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()

	// Create test participants
	participant1 := types.Participant{
		Id:             1,
		Role:           types.ParticipantRole_ECOSYSTEM,
		Did:            "did:example:12345",
		CorporationId:  uint64(1),
		Created:        &nowTime,
		Modified:       &nowTime,
		SchemaId:       1,
		EffectiveFrom:  &nowTime,
		EffectiveUntil: &futureTime,
	}

	participant2 := types.Participant{
		Id:                     2,
		Role:                   types.ParticipantRole_ISSUER,
		Did:                    "did:example:67890",
		CorporationId:          uint64(1),
		Created:                &nowTime,
		Modified:               &nowTime,
		SchemaId:               1,
		EffectiveFrom:          &nowTime,
		EffectiveUntil:         &futureTime,
		ValidatorParticipantId: 1,
	}

	participant3 := types.Participant{
		Id:                     3,
		Role:                   types.ParticipantRole_VERIFIER,
		Did:                    "did:example:verifier",
		CorporationId:          uint64(1),
		Created:                &nowTime,
		Modified:               &nowTime,
		SchemaId:               1,
		EffectiveFrom:          &nowTime,
		EffectiveUntil:         &futureTime,
		ValidatorParticipantId: 1,
	}

	require.NoError(t, k.Participant.Set(ctx, participant1.Id, participant1))
	require.NoError(t, k.Participant.Set(ctx, participant2.Id, participant2))
	require.NoError(t, k.Participant.Set(ctx, participant3.Id, participant3))
	require.NoError(t, k.ParticipantCounter.Set(ctx, 4))

	// Create test participant sessions
	session1 := types.ParticipantSession{
		Id:            "test-session-id-1",
		CorporationId: uint64(1),
		VsOperator:    creatorAddr,
		Created:       &nowTime,
		Modified:      &nowTime,
		SessionRecords: []*types.ParticipantSessionRecord{
			{
				IssuerParticipantId: 1,
				Created:             &nowTime,
			},
		},
	}

	session2 := types.ParticipantSession{
		Id:            "test-session-id-2",
		CorporationId: uint64(1),
		VsOperator:    creatorAddr,
		Created:       &nowTime,
		Modified:      &nowTime,
		SessionRecords: []*types.ParticipantSessionRecord{
			{
				IssuerParticipantId:      1,
				WalletAgentParticipantId: 2,
				Created:                  &nowTime,
			},
		},
	}

	require.NoError(t, k.ParticipantSession.Set(ctx, session1.Id, session1))
	require.NoError(t, k.ParticipantSession.Set(ctx, session2.Id, session2))

	// Export genesis state
	genesisState := participant.ExportGenesis(ctx, k)

	// Verify exported data
	require.Equal(t, uint64(4), genesisState.NextParticipantId)
	require.Len(t, genesisState.Participants, 3)
	require.Len(t, genesisState.ParticipantSessions, 2)

	// Create a new keeper instance
	k2, _, _, _, ctx2, _ := keepertest.ParticipantKeeper(t)

	// Initialize with the exported genesis state
	participant.InitGenesis(ctx2, k2, *genesisState)

	// Verify all data was imported correctly
	participant1Get, err := k2.Participant.Get(ctx2, 1)
	require.NoError(t, err)
	require.Equal(t, participant1.Id, participant1Get.Id)
	require.Equal(t, participant1.Did, participant1Get.Did)
	require.Equal(t, participant1.Role, participant1Get.Role)

	participant2Get, err := k2.Participant.Get(ctx2, 2)
	require.NoError(t, err)
	require.Equal(t, participant2.Id, participant2Get.Id)
	require.Equal(t, participant2.ValidatorParticipantId, participant2Get.ValidatorParticipantId)

	participant3Get, err := k2.Participant.Get(ctx2, 3)
	require.NoError(t, err)
	require.Equal(t, participant3.Id, participant3Get.Id)
	require.Equal(t, participant3.CorporationId, participant3Get.CorporationId)

	counter, err := k2.ParticipantCounter.Get(ctx2)
	require.NoError(t, err)
	require.Equal(t, uint64(4), counter)

	session1Get, err := k2.ParticipantSession.Get(ctx2, "test-session-id-1")
	require.NoError(t, err)
	require.Equal(t, session1.Id, session1Get.Id)
	require.Equal(t, session1.SessionRecords[0].IssuerParticipantId, session1Get.SessionRecords[0].IssuerParticipantId)

	session2Get, err := k2.ParticipantSession.Get(ctx2, "test-session-id-2")
	require.NoError(t, err)
	require.Equal(t, session2.Id, session2Get.Id)
	require.Equal(t, session2.SessionRecords[0].WalletAgentParticipantId, session2Get.SessionRecords[0].WalletAgentParticipantId)

	// Export from the new keeper and verify it matches the original export
	exportedState2 := participant.ExportGenesis(ctx2, k2)

	// Both states should be identical
	nullify.Fill(genesisState)
	nullify.Fill(exportedState2)
	require.Equal(t, genesisState, exportedState2)
}

func TestGenesisValidation(t *testing.T) {
	nowTime := time.Now()
	creatorAddr := sdk.AccAddress([]byte("test_creator")).String()
	_ = creatorAddr

	testCases := []struct {
		name         string
		genesisState types.GenesisState
		expectedErr  string
	}{
		{
			name: "duplicate participant IDs",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Participants: []types.Participant{
					{
						Id:                1,
						Role:              types.ParticipantRole_ISSUER,
						Did:               "did:example:dup1",
						CorporationId:     uint64(1),
						Created:           &nowTime,
						Modified:          &nowTime,
						OpState:           types.OnboardingState_VALIDATED,
						OpLastStateChange: &nowTime,
					},
					{
						Id:            1, // Duplicate ID
						Role:          types.ParticipantRole_VERIFIER,
						CorporationId: uint64(1),
						Created:       &nowTime,
						Modified:      &nowTime,
					},
				},
				NextParticipantId: 2,
			},
			expectedErr: "duplicate participant ID found: 1",
		},
		{
			name: "next participant ID too low",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Participants: []types.Participant{
					{
						Id:                5,
						Role:              types.ParticipantRole_ISSUER,
						Did:               "did:example:five",
						CorporationId:     uint64(1),
						Created:           &nowTime,
						Modified:          &nowTime,
						OpState:           types.OnboardingState_VALIDATED,
						OpLastStateChange: &nowTime,
					},
				},
				NextParticipantId: 3, // Should be > 5
			},
			expectedErr: "next_participant_id (3) must be greater than the maximum participant ID (5)",
		},
		{
			name: "missing required participant field",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Participants: []types.Participant{
					{
						Id:   1,
						Role: types.ParticipantRole_ISSUER,
						// Missing CorporationId field (0)
						Created:  &nowTime,
						Modified: &nowTime,
					},
				},
				NextParticipantId: 2,
			},
			expectedErr: "corporation_id cannot be 0 for participant ID 1",
		},
		{
			name: "invalid validator reference",
			genesisState: types.GenesisState{
				Params: types.DefaultParams(),
				Participants: []types.Participant{
					{
						Id:                     2,
						Role:                   types.ParticipantRole_ISSUER,
						Did:                    "did:example:val2",
						CorporationId:          uint64(1),
						Created:                &nowTime,
						Modified:               &nowTime,
						OpState:                types.OnboardingState_VALIDATED,
						OpLastStateChange:      &nowTime,
						ValidatorParticipantId: 999, // Non-existent validator
					},
				},
				NextParticipantId: 3,
			},
			expectedErr: "validator participant ID 999 not found for participant ID 2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate the genesis state
			err := tc.genesisState.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}
