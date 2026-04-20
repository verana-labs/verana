package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	// Check for duplicate corporations in trust deposits
	corporationSet := make(map[string]struct{}, len(gs.TrustDeposits))

	for i, td := range gs.TrustDeposits {
		if _, err := sdk.AccAddressFromBech32(td.Corporation); err != nil {
			return fmt.Errorf("invalid corporation address at index %d: %s", i, err)
		}

		if _, exists := corporationSet[td.Corporation]; exists {
			return fmt.Errorf("duplicate trust deposit for corporation: %s", td.Corporation)
		}
		corporationSet[td.Corporation] = struct{}{}

		if td.Deposit < td.Claimable {
			return fmt.Errorf("claimable amount exceeds deposit for corporation %s: %d > %d",
				td.Corporation, td.Claimable, td.Deposit)
		}
	}

	return nil
}
