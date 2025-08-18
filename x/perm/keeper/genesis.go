package keeper

import (
	"context"
	"cosmossdk.io/collections"
	"errors"
	"fmt"
	"github.com/verana-labs/verana/x/perm/types"
	"sort"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *types.GenesisState {
	return &types.GenesisState{
		Params:             types.DefaultParams(),
		Permissions:        []types.Permission{},
		PermissionSessions: []types.PermissionSession{},
		NextPermissionId:   1, // Start with 1 as first ID
	}
}

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	// Set module parameters
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	// Import all permissions
	for _, perm := range genState.Permissions {
		if err := k.Permission.Set(ctx, perm.Id, perm); err != nil {
			panic(fmt.Errorf("failed to set perm: %w", err))
		}
	}

	// Import all perm sessions
	for _, session := range genState.PermissionSessions {
		if err := k.PermissionSession.Set(ctx, session.Id, session); err != nil {
			panic(fmt.Errorf("failed to set perm session: %w", err))
		}
	}

	// Set the permissions counter
	if err := k.PermissionCounter.Set(ctx, genState.NextPermissionId); err != nil {
		panic(fmt.Errorf("failed to set perm counter: %w", err))
	}
	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	genesis := DefaultGenesis()

	// Export module parameters
	genesis.Params = k.GetParams(ctx)

	// Export all permissions
	permissions := []types.Permission{}
	if err := k.Permission.Walk(ctx, nil, func(id uint64, perm types.Permission) (bool, error) {
		permissions = append(permissions, perm)
		return false, nil
	}); err != nil {
		panic(fmt.Errorf("failed to export permissions: %w", err))
	}

	// Sort permissions by ID for deterministic output
	sort.Slice(permissions, func(i, j int) bool {
		return permissions[i].Id < permissions[j].Id
	})

	genesis.Permissions = permissions

	// Export all perm sessions
	sessions := []types.PermissionSession{}
	if err := k.PermissionSession.Walk(ctx, nil, func(id string, session types.PermissionSession) (bool, error) {
		sessions = append(sessions, session)
		return false, nil
	}); err != nil {
		panic(fmt.Errorf("failed to export perm sessions: %w", err))
	}

	// Sort sessions by ID for deterministic output
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Id < sessions[j].Id
	})

	genesis.PermissionSessions = sessions

	// Export perm counter
	nextId, err := k.PermissionCounter.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		panic(fmt.Errorf("failed to get perm counter: %w", err))
	}

	// In case of no permissions, set next ID to 1
	if errors.Is(err, collections.ErrNotFound) {
		nextId = 1
	}

	genesis.NextPermissionId = nextId

	return genesis, nil
}
