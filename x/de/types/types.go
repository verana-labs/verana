package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// VPRDelegableMsgTypes is the set of VPR message types that can be delegated
// via operator authorization. CreateOrUpdatePermissionSession is excluded
// per the spec.
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
	// Permission (PERM) - excluding CreateOrUpdatePermissionSession
	"/verana.perm.v1.MsgStartPermissionVP":                       true,
	"/verana.perm.v1.MsgRenewPermissionVP":                       true,
	"/verana.perm.v1.MsgSetPermissionVPToValidated":               true,
	"/verana.perm.v1.MsgCancelPermissionVPLastRequest":            true,
	"/verana.perm.v1.MsgCreateRootPermission":                     true,
	"/verana.perm.v1.MsgExtendPermission":                         true,
	"/verana.perm.v1.MsgRevokePermission":                         true,
	"/verana.perm.v1.MsgSlashPermissionTrustDeposit":              true,
	"/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit":       true,
	"/verana.perm.v1.MsgCreatePermission":                         true,
	// Trust Deposit (TD)
	"/verana.td.v1.MsgReclaimTrustDepositYield":                   true,
	"/verana.td.v1.MsgReclaimTrustDeposit":                        true,
	"/verana.td.v1.MsgRepaySlashedTrustDeposit":                   true,
	// Delegation (DE)
	"/verana.de.v1.MsgGrantOperatorAuthorization":                  true,
	"/verana.de.v1.MsgRevokeOperatorAuthorization":                 true,
}

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

	// msg_types must not be empty and must be VPR delegable
	if len(msg.MsgTypes) == 0 {
		return fmt.Errorf("msg_types must not be empty")
	}
	for _, mt := range msg.MsgTypes {
		if !VPRDelegableMsgTypes[mt] {
			return fmt.Errorf("msg_type %s is not a VPR delegable message type", mt)
		}
	}

	// authz_spend_limit if specified must be valid
	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsValid() {
		return fmt.Errorf("invalid authz_spend_limit")
	}

	// feegrant_spend_limit if specified must be valid (only relevant if with_feegrant)
	if msg.WithFeegrant && len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsValid() {
		return fmt.Errorf("invalid feegrant_spend_limit")
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
