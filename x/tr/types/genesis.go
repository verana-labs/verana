package types

import (
	"fmt"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                       DefaultParams(),
		TrustRegistries:              []TrustRegistry{},
		GovernanceFrameworkVersions:  []GovernanceFrameworkVersion{},
		GovernanceFrameworkDocuments: []GovernanceFrameworkDocument{},
		Counters:                     []Counter{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate params
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Validate trust registries
	seenTrustRegistryIDs := make(map[uint64]bool)
	seenDIDs := make(map[string]bool)
	for _, tr := range gs.TrustRegistries {
		// Check for duplicate IDs
		if seenTrustRegistryIDs[tr.Id] {
			return fmt.Errorf("duplicate trust registry ID found in genesis state: %d", tr.Id)
		}
		seenTrustRegistryIDs[tr.Id] = true

		// Check for duplicate DIDs
		if seenDIDs[tr.Did] {
			return fmt.Errorf("duplicate DID found in genesis state: %s", tr.Did)
		}
		seenDIDs[tr.Did] = true
	}

	// Validate governance framework versions
	seenGFVersionIDs := make(map[uint64]bool)
	for _, gfv := range gs.GovernanceFrameworkVersions {
		// Check for duplicate IDs
		if seenGFVersionIDs[gfv.Id] {
			return fmt.Errorf("duplicate governance framework version ID found in genesis state: %d", gfv.Id)
		}
		seenGFVersionIDs[gfv.Id] = true

		// Check that trust registry exists
		if !seenTrustRegistryIDs[gfv.TrId] {
			return fmt.Errorf("governance framework version references non-existent trust registry ID: %d", gfv.TrId)
		}
	}

	// Validate governance framework documents
	seenGFDocumentIDs := make(map[uint64]bool)
	for _, gfd := range gs.GovernanceFrameworkDocuments {
		// Check for duplicate IDs
		if seenGFDocumentIDs[gfd.Id] {
			return fmt.Errorf("duplicate governance framework document ID found in genesis state: %d", gfd.Id)
		}
		seenGFDocumentIDs[gfd.Id] = true

		// Check that governance framework version exists
		if !seenGFVersionIDs[gfd.GfvId] {
			return fmt.Errorf("governance framework document references non-existent version ID: %d", gfd.GfvId)
		}
	}

	// Validate counters
	counterMap := make(map[string]uint64) // Temporary map for validation
	for _, counter := range gs.Counters {
		// Check for duplicate counter entity types
		if _, exists := counterMap[counter.EntityType]; exists {
			return fmt.Errorf("duplicate counter entity type found: %s", counter.EntityType)
		}
		counterMap[counter.EntityType] = counter.Value

		// Only validate known counter types
		if counter.EntityType != "tr" && counter.EntityType != "gfv" && counter.EntityType != "gfd" {
			return fmt.Errorf("unknown counter entity type: %s", counter.EntityType)
		}

		// Counter values should be valid (e.g., not less than the highest ID of the corresponding entity)
		switch counter.EntityType {
		case "tr":
			for _, tr := range gs.TrustRegistries {
				if tr.Id > counter.Value {
					return fmt.Errorf("trust registry counter (%d) is less than maximum trust registry ID (%d)", counter.Value, tr.Id)
				}
			}
		case "gfv":
			for _, gfv := range gs.GovernanceFrameworkVersions {
				if gfv.Id > counter.Value {
					return fmt.Errorf("governance framework version counter (%d) is less than maximum version ID (%d)", counter.Value, gfv.Id)
				}
			}
		case "gfd":
			for _, gfd := range gs.GovernanceFrameworkDocuments {
				if gfd.Id > counter.Value {
					return fmt.Errorf("governance framework document counter (%d) is less than maximum document ID (%d)", counter.Value, gfd.Id)
				}
			}
		}
	}

	return nil
}
