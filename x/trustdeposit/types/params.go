package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultTrustDepositReclaimBurnRate = "0.6" // 60%
	DefaultTrustDepositShareValue      = "1.0" // Initial value: 1
	DefaultTrustDepositRate            = "0.2" // 20%
	DefaultWalletUserAgentRewardRate   = "0.2" // 20%
	DefaultUserAgentRewardRate         = "0.2" // 20%
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	trustDepositReclaimBurnRate math.LegacyDec,
	trustDepositShareValue math.LegacyDec,
	trustDepositRate math.LegacyDec,
	walletUserAgentRewardRate math.LegacyDec,
	userAgentRewardRate math.LegacyDec,
) Params {
	return Params{
		TrustDepositReclaimBurnRate: trustDepositReclaimBurnRate,
		TrustDepositShareValue:      trustDepositShareValue,
		TrustDepositRate:            trustDepositRate,
		WalletUserAgentRewardRate:   walletUserAgentRewardRate,
		UserAgentRewardRate:         userAgentRewardRate,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	TrustDepositReclaimBurnRate, _ := math.LegacyNewDecFromStr(DefaultTrustDepositReclaimBurnRate)
	TrustDepositShareValue, _ := math.LegacyNewDecFromStr(DefaultTrustDepositShareValue)
	TrustDepositRate, _ := math.LegacyNewDecFromStr(DefaultTrustDepositRate)
	WalletUserAgentRewardRate, _ := math.LegacyNewDecFromStr(DefaultWalletUserAgentRewardRate)
	UserAgentRewardRate, _ := math.LegacyNewDecFromStr(DefaultUserAgentRewardRate)

	return NewParams(
		TrustDepositReclaimBurnRate,
		TrustDepositShareValue,
		TrustDepositRate,
		WalletUserAgentRewardRate,
		UserAgentRewardRate,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositReclaimBurnRate"),
			&p.TrustDepositReclaimBurnRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositShareValue"),
			&p.TrustDepositShareValue,
			validatePositiveLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("TrustDepositRate"),
			&p.TrustDepositRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("WalletUserAgentRewardRate"),
			&p.WalletUserAgentRewardRate,
			validateLegacyDec,
		),
		paramtypes.NewParamSetPair(
			[]byte("UserAgentRewardRate"),
			&p.UserAgentRewardRate,
			validateLegacyDec,
		),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateLegacyDec(p.TrustDepositReclaimBurnRate); err != nil {
		return err
	}
	if err := validatePositiveLegacyDec(p.TrustDepositShareValue); err != nil {
		return err
	}
	if err := validateLegacyDec(p.TrustDepositRate); err != nil {
		return err
	}
	if err := validateLegacyDec(p.WalletUserAgentRewardRate); err != nil {
		return err
	}
	if err := validateLegacyDec(p.UserAgentRewardRate); err != nil {
		return err
	}
	return nil
}

// validateLegacyDec validates that the parameter is a valid decimal between 0 and 1
func validateLegacyDec(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("value cannot be negative: %s", v)
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("value cannot be greater than 1: %s", v)
	}

	return nil
}

// validatePositiveLegacyDec validates that the parameter is a positive decimal
func validatePositiveLegacyDec(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() || v.IsZero() {
		return fmt.Errorf("value must be positive: %s", v)
	}

	return nil
}
