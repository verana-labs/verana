package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "xr"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"

	// CounterKeyExchangeRate is the counter key for exchange rate auto-id
	CounterKeyExchangeRate = "exchange_rate"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_xr")

// ExchangeRateKey is the prefix for exchange rate storage
var ExchangeRateKey = collections.NewPrefix("xr_rate")

// CounterKey is the prefix for counters
var CounterKey = collections.NewPrefix("xr_ctr")

// ExchangeRatePairIndexKey is the prefix for the pair uniqueness index
var ExchangeRatePairIndexKey = collections.NewPrefix("xr_pidx")
