package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/di module sentinel errors
var (
	ErrInvalidSigner       = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDigestEmpty         = errors.Register(ModuleName, 1101, "digest must not be empty")
	ErrDelegationKeeperNil = errors.Register(ModuleName, 1102, "delegation keeper is required but not set")
)
