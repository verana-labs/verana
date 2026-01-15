package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/td module sentinel errors
var (
	ErrInvalidSigner            = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                   = errors.Register(ModuleName, 1101, "sample error")
	ErrorUnknownProposalType    = errors.Register(ModuleName, 1102, "unknown proposal type")
	ErrInvalidAmount            = errors.Register(ModuleName, 1103, "invalid amount")
	ErrTrustDepositNotFound     = errors.Register(ModuleName, 1104, "trust deposit not found")
	ErrInsufficientTrustDeposit = errors.Register(ModuleName, 1105, "insufficient trust deposit")

	// Anchor-based POC errors
	ErrUnauthorized   = errors.Register(ModuleName, 1200, "unauthorized")
	ErrInvalidAddress = errors.Register(ModuleName, 1201, "invalid address")
	ErrAnchorNotFound = errors.Register(ModuleName, 1202, "anchor not found")
)
