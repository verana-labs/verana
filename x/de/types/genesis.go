package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                      DefaultParams(),
		OperatorAuthorizations:      []OperatorAuthorization{},
		FeeGrants:                   []FeeGrant{},
		VsOperatorAuthorizations:    []VSOperatorAuthorization{},
		OperatorAuthorizationUsages: []OperatorAuthorizationUsage{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	for i, usage := range gs.OperatorAuthorizationUsages {
		if _, err := sdk.AccAddressFromBech32(usage.Corporation); err != nil {
			return fmt.Errorf("operator_authorization_usages[%d]: invalid corporation address: %w", i, err)
		}
		if _, err := sdk.AccAddressFromBech32(usage.Operator); err != nil {
			return fmt.Errorf("operator_authorization_usages[%d]: invalid operator address: %w", i, err)
		}
	}

	return nil
}
