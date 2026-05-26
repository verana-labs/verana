package types

import "cosmossdk.io/errors"

var (
	ErrInvalidSigner = errors.Register(ModuleName, 1100, "expected gov account as only signer")

	ErrInvalidSubject       = errors.Register(ModuleName, 1101, "invalid GF subject: must be either ecosystem_id or corporation")
	ErrSubjectNotFound      = errors.Register(ModuleName, 1102, "GF subject not found")
	ErrSubjectNotControlled = errors.Register(ModuleName, 1103, "signing corporation is not the controller of the target subject")
	ErrInvalidVersion       = errors.Register(ModuleName, 1104, "invalid governance framework version")
	ErrInvalidLanguage      = errors.Register(ModuleName, 1105, "invalid BCP 47 language tag")
	ErrInvalidURL           = errors.Register(ModuleName, 1106, "invalid URL")
	ErrInvalidDigestSRI     = errors.Register(ModuleName, 1107, "invalid digest_sri")
	ErrNoActivatableVersion = errors.Register(ModuleName, 1108, "no governance framework version available to activate")
	ErrMissingDefaultLang   = errors.Register(ModuleName, 1109, "no document found for the default language of this version")
)
