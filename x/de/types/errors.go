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
	ErrAuthzNotFound         = errors.Register(ModuleName, 1105, "operator authorization not found for this authority/operator pair")
	ErrAuthzExpired          = errors.Register(ModuleName, 1106, "operator authorization has expired")
	ErrAuthzMsgTypeNotFound  = errors.Register(ModuleName, 1107, "operator authorization does not include requested message type")
	ErrAuthzSpendLimitExceeded = errors.Register(ModuleName, 1108, "operator authorization spend limit exceeded")
	ErrOperatorAuthzNotFound   = errors.Register(ModuleName, 1109, "operator authorization not found for this authority/grantee pair")
)
