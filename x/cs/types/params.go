package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	// Default parameter values
	DefaultCredentialSchemaSchemaMaxSize                                  = uint64(8192) // 8KB
	DefaultCredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays   = uint32(3650) // 10 years
	DefaultCredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays = uint32(3650)
	DefaultCredentialSchemaIssuerValidationValidityPeriodMaxDays          = uint32(3650)
	DefaultCredentialSchemaVerifierValidationValidityPeriodMaxDays        = uint32(3650)
	DefaultCredentialSchemaHolderValidationValidityPeriodMaxDays          = uint32(3650)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	schemaMaxSize uint64,
	issuerGrantorValidityPeriod uint32,
	verifierGrantorValidityPeriod uint32,
	issuerValidityPeriod uint32,
	verifierValidityPeriod uint32,
	holderValidityPeriod uint32,
) Params {
	return Params{
		CredentialSchemaSchemaMaxSize:                                  schemaMaxSize,
		CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays:   issuerGrantorValidityPeriod,
		CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays: verifierGrantorValidityPeriod,
		CredentialSchemaIssuerValidationValidityPeriodMaxDays:          issuerValidityPeriod,
		CredentialSchemaVerifierValidationValidityPeriodMaxDays:        verifierValidityPeriod,
		CredentialSchemaHolderValidationValidityPeriodMaxDays:          holderValidityPeriod,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultCredentialSchemaSchemaMaxSize,
		DefaultCredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays,
		DefaultCredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays,
		DefaultCredentialSchemaIssuerValidationValidityPeriodMaxDays,
		DefaultCredentialSchemaVerifierValidationValidityPeriodMaxDays,
		DefaultCredentialSchemaHolderValidationValidityPeriodMaxDays,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("CredentialSchemaSchemaMaxSize"),
			&p.CredentialSchemaSchemaMaxSize,
			validatePositiveUint64,
		),
		paramtypes.NewParamSetPair(
			[]byte("CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays"),
			&p.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays,
			validatePositiveUint32,
		),
		paramtypes.NewParamSetPair(
			[]byte("CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays"),
			&p.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays,
			validatePositiveUint32,
		),
		paramtypes.NewParamSetPair(
			[]byte("CredentialSchemaIssuerValidationValidityPeriodMaxDays"),
			&p.CredentialSchemaIssuerValidationValidityPeriodMaxDays,
			validatePositiveUint32,
		),
		paramtypes.NewParamSetPair(
			[]byte("CredentialSchemaVerifierValidationValidityPeriodMaxDays"),
			&p.CredentialSchemaVerifierValidationValidityPeriodMaxDays,
			validatePositiveUint32,
		),
		paramtypes.NewParamSetPair(
			[]byte("CredentialSchemaHolderValidationValidityPeriodMaxDays"),
			&p.CredentialSchemaHolderValidationValidityPeriodMaxDays,
			validatePositiveUint32,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if p.CredentialSchemaSchemaMaxSize == 0 {
		return fmt.Errorf("credential schema max size must be positive")
	}
	if p.CredentialSchemaIssuerGrantorValidationValidityPeriodMaxDays == 0 {
		return fmt.Errorf("issuer grantor validation validity period max days must be positive")
	}
	if p.CredentialSchemaVerifierGrantorValidationValidityPeriodMaxDays == 0 {
		return fmt.Errorf("verifier grantor validation validity period max days must be positive")
	}
	if p.CredentialSchemaIssuerValidationValidityPeriodMaxDays == 0 {
		return fmt.Errorf("issuer validation validity period max days must be positive")
	}
	if p.CredentialSchemaVerifierValidationValidityPeriodMaxDays == 0 {
		return fmt.Errorf("verifier validation validity period max days must be positive")
	}
	if p.CredentialSchemaHolderValidationValidityPeriodMaxDays == 0 {
		return fmt.Errorf("holder validation validity period max days must be positive")
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

func validatePositiveUint32(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("value must be positive: %d", v)
	}

	return nil
}
