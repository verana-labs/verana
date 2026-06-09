package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// VPRDelegableMsgTypes is the set of VPR message types that can be delegated
// via operator authorization. Includes CreateOrUpdatePermissionSession for
// VSOA fee grants. Note: CreateOrUpdatePermissionSession is explicitly
// excluded from operator authorization msg_types (see ValidateBasic).
var VPRDelegableMsgTypes = map[string]bool{
	// Corporation (CO)
	"/verana.co.v1.MsgCreateCorporation": true,
	"/verana.co.v1.MsgUpdateCorporation": true,
	// Ecosystem (EC) — renamed from Trust Registry (TR) in v4-rc2 (#305)
	"/verana.ec.v1.MsgCreateEcosystem":  true,
	"/verana.ec.v1.MsgUpdateEcosystem":  true,
	"/verana.ec.v1.MsgArchiveEcosystem": true,
	// Governance Framework (GF) — extracted from TR
	"/verana.gf.v1.MsgAddGovernanceFrameworkDocument":           true,
	"/verana.gf.v1.MsgIncreaseActiveGovernanceFrameworkVersion": true,
	// Credential Schema (CS)
	"/verana.cs.v1.MsgCreateCredentialSchema":                         true,
	"/verana.cs.v1.MsgUpdateCredentialSchema":                         true,
	"/verana.cs.v1.MsgArchiveCredentialSchema":                        true,
	"/verana.cs.v1.MsgCreateSchemaAuthorizationPolicy":                true,
	"/verana.cs.v1.MsgIncreaseActiveSchemaAuthorizationPolicyVersion": true,
	"/verana.cs.v1.MsgRevokeSchemaAuthorizationPolicy":                true,
	// Permission (PERM) - CreateOrUpdatePermissionSession included for VSOA fee grants
	"/verana.pp.v1.MsgStartParticipantOP":                  true,
	"/verana.pp.v1.MsgRenewParticipantOP":                  true,
	"/verana.pp.v1.MsgSetParticipantOPToValidated":         true,
	"/verana.pp.v1.MsgCancelParticipantOPLastRequest":      true,
	"/verana.pp.v1.MsgCreateRootParticipant":               true,
	"/verana.pp.v1.MsgSetParticipantEffectiveUntil":        true,
	"/verana.pp.v1.MsgRevokeParticipant":                   true,
	"/verana.pp.v1.MsgSlashParticipantTrustDeposit":        true,
	"/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit": true,
	"/verana.pp.v1.MsgSelfCreateParticipant":               true,
	"/verana.pp.v1.MsgCreateOrUpdateParticipantSession":    true,
	// Trust Deposit (TD)
	"/verana.td.v1.MsgReclaimTrustDepositYield": true,
	"/verana.td.v1.MsgRepaySlashedTrustDeposit": true,
	// Digest (DI)
	"/verana.di.v1.MsgStoreDigest": true,
	// Delegation (DE)
	"/verana.de.v1.MsgGrantOperatorAuthorization":  true,
	"/verana.de.v1.MsgRevokeOperatorAuthorization": true,
	// Exchange Rate (XR)
	"/verana.xr.v1.MsgUpdateExchangeRate": true,
}

// MsgCreateOrUpdateParticipantSessionTypeURL is the type URL for
// MsgCreateOrUpdateParticipantSession. Used to exclude it from operator
// authorization msg_types while still allowing it in VSOA fee grants.
const MsgCreateOrUpdateParticipantSessionTypeURL = "/verana.pp.v1.MsgCreateOrUpdateParticipantSession"

// ValidateBasic performs stateless validation on MsgGrantOperatorAuthorization.
func (msg *MsgGrantOperatorAuthorization) ValidateBasic() error {
	// corporation is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// operator is optional; if present, must be valid
	if msg.Operator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
			return fmt.Errorf("invalid operator address: %w", err)
		}
	}

	// grantee is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}

	// msg_types must not be empty and must be VPR delegable
	if len(msg.MsgTypes) == 0 {
		return fmt.Errorf("msg_types must not be empty")
	}
	for _, mt := range msg.MsgTypes {
		if !VPRDelegableMsgTypes[mt] {
			return fmt.Errorf("msg_type %s is not a VPR delegable message type", mt)
		}
		if mt == MsgCreateOrUpdateParticipantSessionTypeURL {
			return fmt.Errorf("msg_type %s is not allowed in operator authorization", mt)
		}
	}

	// authz_spend_limit if specified must be valid and all-positive
	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsValid() {
		return fmt.Errorf("invalid authz_spend_limit")
	}
	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsAllPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "authz_spend_limit amounts must be positive")
	}

	// authz_spend_limit_period if specified must be a valid (positive) period;
	// ignored if authz_spend_limit is not set [MOD-DE-MSG-3-2]
	if len(msg.AuthzSpendLimit) > 0 && msg.AuthzSpendLimitPeriod != nil && *msg.AuthzSpendLimitPeriod <= 0 {
		return fmt.Errorf("authz_spend_limit_period must be a positive duration")
	}

	// feegrant fields must be empty when with_feegrant is false
	if !msg.WithFeegrant {
		if !msg.FeegrantSpendLimit.IsZero() {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "feegrant_spend_limit must be empty when with_feegrant is false")
		}
	}

	// feegrant_spend_limit if specified must be valid and all-positive (only relevant if with_feegrant)
	if msg.WithFeegrant && len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsValid() {
		return fmt.Errorf("invalid feegrant_spend_limit")
	}
	if len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsAllPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "feegrant_spend_limit amounts must be positive")
	}

	// feegrant_spend_limit_period if specified must be a valid (positive) period;
	// ignored if feegrant_spend_limit is not set or with_feegrant is false [MOD-DE-MSG-3-2]
	if msg.WithFeegrant && len(msg.FeegrantSpendLimit) > 0 && msg.FeegrantSpendLimitPeriod != nil && *msg.FeegrantSpendLimitPeriod <= 0 {
		return fmt.Errorf("feegrant_spend_limit_period must be a positive duration")
	}

	return nil
}

// ValidateBasic performs stateless validation on MsgRevokeOperatorAuthorization.
func (msg *MsgRevokeOperatorAuthorization) ValidateBasic() error {
	// corporation is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// operator is optional; if present, must be valid
	if msg.Operator != "" {
		if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
			return fmt.Errorf("invalid operator address: %w", err)
		}
	}

	// grantee is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}

	return nil
}
