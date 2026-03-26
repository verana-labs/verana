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
)
