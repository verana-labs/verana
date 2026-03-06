/**
 * Custom Registry for Verana blockchain message types.
 * Registers all Verana-specific protobuf message types with CosmJS.
 */

import { Registry, GeneratedType } from "@cosmjs/proto-signing";
import { defaultRegistryTypes } from "@cosmjs/stargate";

// Trust Registry (tr) module messages
import {
  MsgCreateTrustRegistry,
  MsgUpdateTrustRegistry,
  MsgArchiveTrustRegistry,
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "../../../src/codec/verana/tr/v1/tx";

// DID Directory (dd) module messages
import {
  MsgAddDID,
  MsgRenewDID,
  MsgRemoveDID,
  MsgTouchDID,
} from "../../../src/codec/verana/dd/v1/tx";

// Credential Schema (cs) module messages
import {
  MsgCreateCredentialSchema,
  MsgUpdateCredentialSchema,
  MsgArchiveCredentialSchema,
} from "../../../src/codec/verana/cs/v1/tx";

// Permission (perm) module messages
import {
  MsgCreatePermission,
  MsgCreateRootPermission,
  MsgExtendPermission,
  MsgRevokePermission,
  MsgStartPermissionVP,
  MsgRenewPermissionVP,
  MsgSetPermissionVPToValidated,
  MsgCancelPermissionVPLastRequest,
  MsgCreateOrUpdatePermissionSession,
  MsgSlashPermissionTrustDeposit,
  MsgRepayPermissionSlashedTrustDeposit,
} from "../../../src/codec/verana/perm/v1/tx";

// Trust Deposit (td) module messages
import {
  MsgReclaimTrustDeposit,
  MsgReclaimTrustDepositYield,
  MsgSlashTrustDeposit,
  MsgRepaySlashedTrustDeposit,
} from "../../../src/codec/verana/td/v1/tx";

// Delegation Engine (de) module messages
import {
  MsgGrantOperatorAuthorization,
  MsgRevokeOperatorAuthorization,
} from "../../../src/codec/verana/de/v1/tx";

// Type URLs for all Verana messages
export const typeUrls = {
  // Trust Registry
  MsgCreateTrustRegistry: "/verana.tr.v1.MsgCreateTrustRegistry",
  MsgUpdateTrustRegistry: "/verana.tr.v1.MsgUpdateTrustRegistry",
  MsgArchiveTrustRegistry: "/verana.tr.v1.MsgArchiveTrustRegistry",
  MsgAddGovernanceFrameworkDocument: "/verana.tr.v1.MsgAddGovernanceFrameworkDocument",
  MsgIncreaseActiveGovernanceFrameworkVersion: "/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",

  // DID Directory
  MsgAddDID: "/verana.dd.v1.MsgAddDID",
  MsgRenewDID: "/verana.dd.v1.MsgRenewDID",
  MsgRemoveDID: "/verana.dd.v1.MsgRemoveDID",
  MsgTouchDID: "/verana.dd.v1.MsgTouchDID",

  // Credential Schema
  MsgCreateCredentialSchema: "/verana.cs.v1.MsgCreateCredentialSchema",
  MsgUpdateCredentialSchema: "/verana.cs.v1.MsgUpdateCredentialSchema",
  MsgArchiveCredentialSchema: "/verana.cs.v1.MsgArchiveCredentialSchema",

  // Permission
  MsgCreatePermission: "/verana.perm.v1.MsgCreatePermission",
  MsgCreateRootPermission: "/verana.perm.v1.MsgCreateRootPermission",
  MsgExtendPermission: "/verana.perm.v1.MsgExtendPermission",
  MsgRevokePermission: "/verana.perm.v1.MsgRevokePermission",
  MsgStartPermissionVP: "/verana.perm.v1.MsgStartPermissionVP",
  MsgRenewPermissionVP: "/verana.perm.v1.MsgRenewPermissionVP",
  MsgSetPermissionVPToValidated: "/verana.perm.v1.MsgSetPermissionVPToValidated",
  MsgCancelPermissionVPLastRequest: "/verana.perm.v1.MsgCancelPermissionVPLastRequest",
  MsgCreateOrUpdatePermissionSession: "/verana.perm.v1.MsgCreateOrUpdatePermissionSession",
  MsgSlashPermissionTrustDeposit: "/verana.perm.v1.MsgSlashPermissionTrustDeposit",
  MsgRepayPermissionSlashedTrustDeposit: "/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit",

  // Trust Deposit
  MsgReclaimTrustDeposit: "/verana.td.v1.MsgReclaimTrustDeposit",
  MsgReclaimTrustDepositYield: "/verana.td.v1.MsgReclaimTrustDepositYield",
  MsgSlashTrustDeposit: "/verana.td.v1.MsgSlashTrustDeposit",
  MsgRepaySlashedTrustDeposit: "/verana.td.v1.MsgRepaySlashedTrustDeposit",

  // Delegation Engine
  MsgGrantOperatorAuthorization: "/verana.de.v1.MsgGrantOperatorAuthorization",
  MsgRevokeOperatorAuthorization: "/verana.de.v1.MsgRevokeOperatorAuthorization",
} as const;

/**
 * Creates a Registry with all Verana custom message types registered.
 */
export function createVeranaRegistry(): Registry {
  const registry = new Registry(defaultRegistryTypes);

  // Trust Registry messages
  registry.register(typeUrls.MsgCreateTrustRegistry, MsgCreateTrustRegistry as GeneratedType);
  registry.register(typeUrls.MsgUpdateTrustRegistry, MsgUpdateTrustRegistry as GeneratedType);
  registry.register(typeUrls.MsgArchiveTrustRegistry, MsgArchiveTrustRegistry as GeneratedType);
  registry.register(typeUrls.MsgAddGovernanceFrameworkDocument, MsgAddGovernanceFrameworkDocument as GeneratedType);
  registry.register(typeUrls.MsgIncreaseActiveGovernanceFrameworkVersion, MsgIncreaseActiveGovernanceFrameworkVersion as GeneratedType);

  // DID Directory messages
  registry.register(typeUrls.MsgAddDID, MsgAddDID as GeneratedType);
  registry.register(typeUrls.MsgRenewDID, MsgRenewDID as GeneratedType);
  registry.register(typeUrls.MsgRemoveDID, MsgRemoveDID as GeneratedType);
  registry.register(typeUrls.MsgTouchDID, MsgTouchDID as GeneratedType);

  // Credential Schema messages
  registry.register(typeUrls.MsgCreateCredentialSchema, MsgCreateCredentialSchema as GeneratedType);
  registry.register(typeUrls.MsgUpdateCredentialSchema, MsgUpdateCredentialSchema as GeneratedType);
  registry.register(typeUrls.MsgArchiveCredentialSchema, MsgArchiveCredentialSchema as GeneratedType);

  // Permission messages
  registry.register(typeUrls.MsgCreatePermission, MsgCreatePermission as GeneratedType);
  registry.register(typeUrls.MsgCreateRootPermission, MsgCreateRootPermission as GeneratedType);
  registry.register(typeUrls.MsgExtendPermission, MsgExtendPermission as GeneratedType);
  registry.register(typeUrls.MsgRevokePermission, MsgRevokePermission as GeneratedType);
  registry.register(typeUrls.MsgStartPermissionVP, MsgStartPermissionVP as GeneratedType);
  registry.register(typeUrls.MsgRenewPermissionVP, MsgRenewPermissionVP as GeneratedType);
  registry.register(typeUrls.MsgSetPermissionVPToValidated, MsgSetPermissionVPToValidated as GeneratedType);
  registry.register(typeUrls.MsgCancelPermissionVPLastRequest, MsgCancelPermissionVPLastRequest as GeneratedType);
  registry.register(typeUrls.MsgCreateOrUpdatePermissionSession, MsgCreateOrUpdatePermissionSession as GeneratedType);
  registry.register(typeUrls.MsgSlashPermissionTrustDeposit, MsgSlashPermissionTrustDeposit as GeneratedType);
  registry.register(typeUrls.MsgRepayPermissionSlashedTrustDeposit, MsgRepayPermissionSlashedTrustDeposit as GeneratedType);

  // Trust Deposit messages
  registry.register(typeUrls.MsgReclaimTrustDeposit, MsgReclaimTrustDeposit as GeneratedType);
  registry.register(typeUrls.MsgReclaimTrustDepositYield, MsgReclaimTrustDepositYield as GeneratedType);
  registry.register(typeUrls.MsgSlashTrustDeposit, MsgSlashTrustDeposit as GeneratedType);
  registry.register(typeUrls.MsgRepaySlashedTrustDeposit, MsgRepaySlashedTrustDeposit as GeneratedType);

  // Delegation Engine messages
  registry.register(typeUrls.MsgGrantOperatorAuthorization, MsgGrantOperatorAuthorization as GeneratedType);
  registry.register(typeUrls.MsgRevokeOperatorAuthorization, MsgRevokeOperatorAuthorization as GeneratedType);

  return registry;
}
