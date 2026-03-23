package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VPRDelegableMsgTypes is the set of VPR message types that can be used in
// fee grants and VS operator authorizations.
var VPRDelegableMsgTypes = map[string]bool{
	// Trust Registry (TR)
	"/verana.tr.v1.MsgCreateTrustRegistry":                       true,
	"/verana.tr.v1.MsgAddGovernanceFrameworkDocument":             true,
	"/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion":   true,
	"/verana.tr.v1.MsgUpdateTrustRegistry":                       true,
	"/verana.tr.v1.MsgArchiveTrustRegistry":                      true,
	// Credential Schema (CS)
	"/verana.cs.v1.MsgCreateCredentialSchema":                     true,
	"/verana.cs.v1.MsgUpdateCredentialSchema":                     true,
	"/verana.cs.v1.MsgArchiveCredentialSchema":                    true,
	// Permission (PERM)
	"/verana.perm.v1.MsgStartPermissionVP":                       true,
	"/verana.perm.v1.MsgRenewPermissionVP":                       true,
	"/verana.perm.v1.MsgSetPermissionVPToValidated":               true,
	"/verana.perm.v1.MsgCancelPermissionVPLastRequest":            true,
	"/verana.perm.v1.MsgCreateRootPermission":                     true,
	"/verana.perm.v1.MsgAdjustPermission":                         true,
	"/verana.perm.v1.MsgRevokePermission":                         true,
	"/verana.perm.v1.MsgSlashPermissionTrustDeposit":              true,
	"/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit":       true,
	"/verana.perm.v1.MsgCreatePermission":                         true,
	"/verana.perm.v1.MsgCreateOrUpdatePermissionSession":          true,
	// Trust Deposit (TD)
	"/verana.td.v1.MsgReclaimTrustDepositYield":                   true,
	"/verana.td.v1.MsgReclaimTrustDeposit":                        true,
	"/verana.td.v1.MsgRepaySlashedTrustDeposit":                   true,
	// Delegation (DE)
	"/verana.de.v1.MsgGrantOperatorAuthorization":                  true,
	"/verana.de.v1.MsgRevokeOperatorAuthorization":                 true,
}

// MsgCreateOrUpdatePermissionSessionTypeURL is excluded from Operator Authorization
// msg_types per [MOD-DE-MSG-3-2] but allowed in VS Operator fee grants [MOD-DE-MSG-5].
const MsgCreateOrUpdatePermissionSessionTypeURL = "/verana.perm.v1.MsgCreateOrUpdatePermissionSession"

// ValidateBasic performs stateless validation on MsgGrantOperatorAuthorization.
func (msg *MsgGrantOperatorAuthorization) ValidateBasic() error {
	// authority is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
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

	// msg_types must not be empty and must be VPR delegable,
	// excluding CreateOrUpdatePermissionSession per [MOD-DE-MSG-3-2]
	if len(msg.MsgTypes) == 0 {
		return fmt.Errorf("msg_types must not be empty")
	}
	for _, mt := range msg.MsgTypes {
		if !VPRDelegableMsgTypes[mt] {
			return fmt.Errorf("msg_type %s is not a VPR delegable message type", mt)
		}
		if mt == MsgCreateOrUpdatePermissionSessionTypeURL {
			return fmt.Errorf("msg_type %s is not allowed in operator authorization", mt)
		}
	}

	// authz_spend_limit if specified must be valid
	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsValid() {
		return fmt.Errorf("invalid authz_spend_limit")
	}

	// authz_spend_limit_period if specified must be a valid (positive) period;
	// ignored if authz_spend_limit is not set [MOD-DE-MSG-3-2]
	if len(msg.AuthzSpendLimit) > 0 && msg.AuthzSpendLimitPeriod != nil && *msg.AuthzSpendLimitPeriod <= 0 {
		return fmt.Errorf("authz_spend_limit_period must be a positive duration")
	}

	// feegrant_spend_limit if specified must be valid (only relevant if with_feegrant)
	if msg.WithFeegrant && len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsValid() {
		return fmt.Errorf("invalid feegrant_spend_limit")
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
	// authority is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
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
