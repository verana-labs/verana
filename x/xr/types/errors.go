package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/xr module sentinel errors
var (
	ErrInvalidSigner    = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidAssetType = errors.Register(ModuleName, 1101, "invalid pricing asset type")
	ErrInvalidAsset     = errors.Register(ModuleName, 1102, "invalid asset value")
	ErrInvalidRate      = errors.Register(ModuleName, 1103, "invalid rate")
	ErrInvalidRateScale = errors.Register(ModuleName, 1104, "rate_scale must be <= 18")
	ErrInvalidDuration  = errors.Register(ModuleName, 1105, "validity_duration must be >= 1 minute")
	ErrDuplicatePair    = errors.Register(ModuleName, 1106, "exchange rate pair already exists")
	ErrIdenticalPair          = errors.Register(ModuleName, 1107, "base and quote asset pair must not be identical")
	ErrExchangeRateNotFound   = errors.Register(ModuleName, 1108, "exchange rate not found")
	ErrExchangeRateNotActive  = errors.Register(ModuleName, 1109, "exchange rate is not active")
	ErrExchangeRateExpired    = errors.Register(ModuleName, 1110, "exchange rate is expired")
	ErrInvalidAmount          = errors.Register(ModuleName, 1111, "invalid amount")
	ErrInvalidRequest         = errors.Register(ModuleName, 1112, "invalid request")
	ErrAuthorizationNotFound  = errors.Register(ModuleName, 1113, "exchange rate authorization not found")
	ErrAuthorizationExpired   = errors.Register(ModuleName, 1114, "exchange rate authorization is expired")
	ErrUpdateTooSoon          = errors.Register(ModuleName, 1115, "exchange rate updated too soon (min_interval not elapsed)")
	ErrRateDeviationExceeded  = errors.Register(ModuleName, 1116, "exchange rate change exceeds max_deviation_bps")
	ErrInvalidExpiration      = errors.Register(ModuleName, 1117, "expiration must be in the future")
	ErrInvalidMaxDeviation    = errors.Register(ModuleName, 1118, "max_deviation_bps must be in range (0, 10000]")
	ErrInvalidMinInterval     = errors.Register(ModuleName, 1119, "min_interval must be a strictly positive duration")
)
