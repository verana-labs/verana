package types

import "time"

const DefaultMaxValidityDuration = 365 * 24 * time.Hour

// NewParams creates a new Params instance.
func NewParams() Params {
	return Params{
		MaxValidityDuration: DefaultMaxValidityDuration,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams()
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if p.MaxValidityDuration <= 0 {
		return ErrInvalidRequest.Wrap("max_validity_duration must be positive")
	}
	return nil
}
