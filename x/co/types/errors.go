package types

import "cosmossdk.io/errors"

var (
	ErrInvalidSigner             = errors.Register(ModuleName, 1100, "expected gov account as only signer")
	ErrCorporationNotRegistered  = errors.Register(ModuleName, 1101, "signing account is not the policy_address of a registered Corporation")
	ErrDIDAlreadyExists          = errors.Register(ModuleName, 1102, "DID is already registered by another Corporation")
	ErrPolicyAddressAlreadyBound = errors.Register(ModuleName, 1103, "policy_address is already bound to an existing Corporation")
	ErrInvalidLanguage           = errors.Register(ModuleName, 1104, "invalid BCP 47 language tag")
	ErrInvalidURL                = errors.Register(ModuleName, 1105, "invalid URL")
	ErrInvalidDigestSRI          = errors.Register(ModuleName, 1106, "invalid digest_sri")
	ErrInvalidDecisionPolicy     = errors.Register(ModuleName, 1107, "invalid decision_policy")
	ErrInvalidMembers            = errors.Register(ModuleName, 1108, "invalid members list")
	ErrInvalidDID                = errors.Register(ModuleName, 1109, "invalid DID syntax")
	ErrCorporationNotFound       = errors.Register(ModuleName, 1110, "corporation not found")
	ErrInvalidTimestamp          = errors.Register(ModuleName, 1111, "invalid corporation timestamp")
)
