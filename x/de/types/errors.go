package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/de module sentinel errors
var (
	ErrInvalidSigner            = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidMsgType           = errors.Register(ModuleName, 1101, "invalid or non-delegable message type")
	ErrExpirationInPast         = errors.Register(ModuleName, 1102, "expiration must be in the future")
	ErrVSOperatorAuthzExists    = errors.Register(ModuleName, 1103, "VSOperatorAuthorization already exists for this corporation/grantee pair; mutual exclusivity violated")
	ErrInvalidSpendLimit        = errors.Register(ModuleName, 1104, "invalid spend limit")
	ErrAuthzNotFound            = errors.Register(ModuleName, 1105, "operator authorization not found for this corporation/operator pair")
	ErrAuthzExpired             = errors.Register(ModuleName, 1106, "operator authorization has expired")
	ErrAuthzMsgTypeNotFound     = errors.Register(ModuleName, 1107, "operator authorization does not include requested message type")
	ErrAuthzSpendLimitExceeded  = errors.Register(ModuleName, 1108, "operator authorization spend limit exceeded")
	ErrOperatorAuthzNotFound    = errors.Register(ModuleName, 1109, "operator authorization not found for this corporation/grantee pair")
	ErrOperatorAuthzExistsMutex = errors.Register(ModuleName, 1110, "OperatorAuthorization already exists for this corporation/vs_operator pair; mutual exclusivity violated")
	ErrInvalidResponseMaxSize   = errors.Register(ModuleName, 1111, "response_max_size must be between 1 and 1024")
	ErrParticipantRecordExists  = errors.Register(ModuleName, 1112, "a ParticipantAuthorizationRecord already exists for this participant_id; must be globally unique")
	ErrVSOAOtherCorporation     = errors.Register(ModuleName, 1113, "vs_operator already has a VSOperatorAuthorization from a different corporation; single-corp constraint violated")
	ErrVSOperatorAuthzNotFound  = errors.Register(ModuleName, 1114, "VS operator authorization not found")
	ErrVSOFeegrantNotEnabled    = errors.Register(ModuleName, 1115, "VS operator authorization record does not enable fee grant")
)
