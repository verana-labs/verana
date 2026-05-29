package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

const (
	DefaultTrustUnitPrice = uint64(1000000) // 1.0 token in utoken representation
)

func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(trustUnitPrice uint64) Params {
	return Params{TrustUnitPrice: trustUnitPrice}
}

func DefaultParams() Params {
	return NewParams(DefaultTrustUnitPrice)
}

func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair([]byte("TrustUnitPrice"), &p.TrustUnitPrice, validatePositiveUint64),
	}
}

func (p Params) Validate() error {
	if p.TrustUnitPrice == 0 {
		return fmt.Errorf("trust unit price must be positive")
	}
	return nil
}

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
