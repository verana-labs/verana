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
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate parameters
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Check for duplicate addresses in trust deposits
	accountSet := make(map[string]struct{}, len(gs.TrustDeposits))

	for i, td := range gs.TrustDeposits {
		// Check for valid account address
		if _, err := sdk.AccAddressFromBech32(td.Account); err != nil {
			return fmt.Errorf("invalid account address at index %d: %s", i, err)
		}

		// Check for duplicate account addresses
		if _, exists := accountSet[td.Account]; exists {
			return fmt.Errorf("duplicate trust deposit for account: %s", td.Account)
		}
		accountSet[td.Account] = struct{}{}

		// Ensure other fields are valid
		if td.Amount < td.Claimable {
			return fmt.Errorf("claimable amount exceeds deposit amount for account %s: %d > %d",
				td.Account, td.Claimable, td.Amount)
		}
	}

	return nil
}
