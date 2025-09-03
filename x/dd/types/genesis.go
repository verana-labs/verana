package types

import (
	"fmt"
	"sort"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		DidDirectories: []DIDDirectory{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate did directories
	seenDIDs := make(map[string]bool)

	for i, didEntry := range gs.DidDirectories {
		// Check for required fields
		if didEntry.Did == "" {
			return fmt.Errorf("empty DID at index %d", i)
		}

		if didEntry.Controller == "" {
			return fmt.Errorf("empty controller for DID %s", didEntry.Did)
		}

		// Check for duplicate DIDs
		if seenDIDs[didEntry.Did] {
			return fmt.Errorf("duplicate DID Directory found in genesis state: %s", didEntry.Did)
		}
		seenDIDs[didEntry.Did] = true

		// Check for valid DID syntax (simplified for genesis validation)
		if !isValidDIDFormat(didEntry.Did) {
			return fmt.Errorf("invalid DID format: %s", didEntry.Did)
		}

		// Validate timestamps
		if didEntry.Created.IsZero() {
			return fmt.Errorf("DID %s has zero created timestamp", didEntry.Did)
		}

		if didEntry.Modified.IsZero() {
			return fmt.Errorf("DID %s has zero modified timestamp", didEntry.Did)
		}

		if didEntry.Exp.IsZero() {
			return fmt.Errorf("DID %s has zero expiration timestamp", didEntry.Did)
		}

		// Modified shouldn't be before Created
		if didEntry.Modified.Before(didEntry.Created) {
			return fmt.Errorf("DID %s has modified timestamp before created timestamp", didEntry.Did)
		}

		// Check that expiration is reasonable (after creation)
		if didEntry.Exp.Before(didEntry.Created) {
			return fmt.Errorf("DID %s has expiration before created timestamp", didEntry.Did)
		}

		// Check for reasonable deposit value
		if didEntry.Deposit <= 0 {
			return fmt.Errorf("DID %s has non-positive deposit value: %d", didEntry.Did, didEntry.Deposit)
		}
	}

	return nil
}

// isValidDIDFormat performs a basic validation of DID format for genesis validation
func isValidDIDFormat(did string) bool {
	// Basic check that it has 'did:' prefix and at least one more segment
	// We'll use a simplified format validation for genesis
	parts := 0
	for i := 0; i < len(did); i++ {
		if did[i] == ':' {
			parts++
		}
	}
	return len(did) >= 8 && did[0:4] == "did:" && parts >= 2
}

// SanitizeGenesisState sorts all DID entries to ensure deterministic ordering
func SanitizeGenesisState(genesisState *GenesisState) *GenesisState {
	// Sort DID directories by DID to ensure deterministic ordering
	sort.SliceStable(genesisState.DidDirectories, func(i, j int) bool {
		return genesisState.DidDirectories[i].Did < genesisState.DidDirectories[j].Did
	})

	return genesisState
}
