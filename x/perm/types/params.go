package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultValidationTermRequestedTimeoutDays = uint64(7) // 7 days
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	validationTermRequestedTimeoutDays uint64,
) Params {
	return Params{
		ValidationTermRequestedTimeoutDays: validationTermRequestedTimeoutDays,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultValidationTermRequestedTimeoutDays,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("ValidationTermRequestedTimeoutDays"),
			&p.ValidationTermRequestedTimeoutDays,
			validatePositiveUint64,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.ValidationTermRequestedTimeoutDays == 0 {
		return fmt.Errorf("validation term requested timeout days must be positive")
	}
	return nil
}

// Parameter validation helpers
func validatePositiveUint64(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("value must be positive: %d", v)
	}

	return nil
}
