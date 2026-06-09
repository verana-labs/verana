package types

import (
	"fmt"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:        DefaultParams(),
		TrustDeposits: []TrustDepositRecord{},
		Dust:          "",
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate parameters
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Check for duplicate corporation_ids in trust deposits
	corporationSet := make(map[uint64]struct{}, len(gs.TrustDeposits))

	for i, td := range gs.TrustDeposits {
		if td.CorporationId == 0 {
			return fmt.Errorf("invalid corporation_id (zero) at index %d", i)
		}

		if _, exists := corporationSet[td.CorporationId]; exists {
			return fmt.Errorf("duplicate trust deposit for corporation_id: %d", td.CorporationId)
		}
		corporationSet[td.CorporationId] = struct{}{}

		if td.Deposit < td.Refunded {
			return fmt.Errorf("refunded amount exceeds deposit for corporation_id %d: %d > %d",
				td.CorporationId, td.Refunded, td.Deposit)
		}
	}

	return nil
}
