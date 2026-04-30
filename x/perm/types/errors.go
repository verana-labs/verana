package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/perm module sentinel errors
var (
	ErrInvalidSigner               = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrSample                      = sdkerrors.Register(ModuleName, 1101, "sample error")
	ErrPermissionNotActive         = sdkerrors.Register(ModuleName, 1102, "permission is not active")
	ErrPermissionDIDEmpty          = sdkerrors.Register(ModuleName, 1103, "permission did is empty")
	ErrTriggerResolverUnauthorized = sdkerrors.Register(ModuleName, 1104, "trigger resolver: signer not authorized for permission")
)
