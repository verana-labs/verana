package participant

import (
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana/x/pp/keeper"
	"github.com/verana-labs/verana/x/pp/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *types.GenesisState {
	return &types.GenesisState{
		Params:              types.DefaultParams(),
		Participants:        []types.Participant{},
		ParticipantSessions: []types.ParticipantSession{},
		NextParticipantId:   1, // Start with 1 as first ID
	}
}

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	// Import all participants
	for _, participant := range genState.Participants {
		if err := k.Participant.Set(ctx, participant.Id, participant); err != nil {
			panic(fmt.Errorf("failed to set participant: %w", err))
		}
	}

	// Import all participant sessions
	for _, session := range genState.ParticipantSessions {
		if err := k.ParticipantSession.Set(ctx, session.Id, session); err != nil {
			panic(fmt.Errorf("failed to set participant session: %w", err))
		}
	}

	// Set the participants counter
	if err := k.ParticipantCounter.Set(ctx, genState.NextParticipantId); err != nil {
		panic(fmt.Errorf("failed to set participant counter: %w", err))
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := DefaultGenesis()

	// Export module parameters
	genesis.Params = k.GetParams(ctx)

	// Export all participants
	participants := []types.Participant{}
	if err := k.Participant.Walk(ctx, nil, func(id uint64, participant types.Participant) (bool, error) {
		participants = append(participants, participant)
		return false, nil
	}); err != nil {
		panic(fmt.Errorf("failed to export participants: %w", err))
	}

	// Sort participants by ID for deterministic output
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Id < participants[j].Id
	})

	genesis.Participants = participants

	// Export all participant sessions
	sessions := []types.ParticipantSession{}
	if err := k.ParticipantSession.Walk(ctx, nil, func(id string, session types.ParticipantSession) (bool, error) {
		sessions = append(sessions, session)
		return false, nil
	}); err != nil {
		panic(fmt.Errorf("failed to export participant sessions: %w", err))
	}

	// Sort sessions by ID for deterministic output
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Id < sessions[j].Id
	})

	genesis.ParticipantSessions = sessions

	// Export participant counter
	nextId, err := k.ParticipantCounter.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		panic(fmt.Errorf("failed to get participant counter: %w", err))
	}

	// In case of no participants, set next ID to 1
	if errors.Is(err, collections.ErrNotFound) {
		nextId = 1
	}

	genesis.NextParticipantId = nextId

	return genesis
}
