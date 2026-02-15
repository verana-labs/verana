package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/de module sentinel errors
var (
	ErrInvalidSigner         = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidMsgType        = errors.Register(ModuleName, 1101, "invalid or non-delegable message type")
	ErrExpirationInPast      = errors.Register(ModuleName, 1102, "expiration must be in the future")
	ErrVSOperatorAuthzExists = errors.Register(ModuleName, 1103, "VSOperatorAuthorization already exists for this authority/grantee pair; mutual exclusivity violated")
	ErrInvalidSpendLimit     = errors.Register(ModuleName, 1104, "invalid spend limit")
)
