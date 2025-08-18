package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/cs module sentinel errors
var (
	ErrInvalidSigner            = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                   = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrCredentialSchemaNotFound = sdkerrors.Register(ModuleName, 1102, "credential schema not found")
	ErrInvalidJSONSchema        = sdkerrors.Register(ModuleName, 1103, "invalid JSON schema")
)
