import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { AminoTypes, defaultRegistryTypes } from "@cosmjs/stargate";
import {
  MsgCreateTrustRegistry,
  MsgUpdateTrustRegistry,
  MsgArchiveTrustRegistry,
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "./codec/verana/tr/v1/tx";
import {
  MsgAddDID,
  MsgRenewDID,
  MsgRemoveDID,
  MsgTouchDID,
} from "./codec/verana/dd/v1/tx";
import {
  MsgCreateCredentialSchema,
  MsgUpdateCredentialSchema,
  MsgArchiveCredentialSchema,
} from "./codec/verana/cs/v1/tx";
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
} from "./codec/verana/perm/v1/tx";
import {
  MsgReclaimTrustDeposit,
  MsgReclaimTrustDepositYield,
  MsgSlashTrustDeposit,
  MsgRepaySlashedTrustDeposit,
} from "./codec/verana/td/v1/tx";
import {
  MsgCreateTrustRegistryAminoConverter,
  MsgUpdateTrustRegistryAminoConverter,
  MsgArchiveTrustRegistryAminoConverter,
  MsgAddGovernanceFrameworkDocumentAminoConverter,
  MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
} from "./amino-converter/tr";
import {
  MsgAddDIDAminoConverter,
  MsgRenewDIDAminoConverter,
  MsgTouchDIDAminoConverter,
  MsgRemoveDIDAminoConverter,
} from "./amino-converter/dd";
import {
  MsgCreateCredentialSchemaAminoConverter,
  MsgUpdateCredentialSchemaAminoConverter,
  MsgArchiveCredentialSchemaAminoConverter,
} from "./amino-converter/cs";
import {
  MsgReclaimTrustDepositAminoConverter,
  MsgReclaimTrustDepositYieldAminoConverter,
  MsgRepaySlashedTrustDepositAminoConverter,
} from "./amino-converter/td";
import {
  MsgCreateRootPermissionAminoConverter,
  MsgCreatePermissionAminoConverter,
  MsgExtendPermissionAminoConverter,
  MsgRevokePermissionAminoConverter,
  MsgStartPermissionVPAminoConverter,
  MsgRenewPermissionVPAminoConverter,
  MsgSetPermissionVPToValidatedAminoConverter,
  MsgCancelPermissionVPLastRequestAminoConverter,
  MsgCreateOrUpdatePermissionSessionAminoConverter,
  MsgSlashPermissionTrustDepositAminoConverter,
  MsgRepayPermissionSlashedTrustDepositAminoConverter,
} from "./amino-converter/perm";

export const veranaTypeUrls = {
  MsgCreateTrustRegistry: "/verana.tr.v1.MsgCreateTrustRegistry",
  MsgUpdateTrustRegistry: "/verana.tr.v1.MsgUpdateTrustRegistry",
  MsgArchiveTrustRegistry: "/verana.tr.v1.MsgArchiveTrustRegistry",
  MsgAddGovernanceFrameworkDocument: "/verana.tr.v1.MsgAddGovernanceFrameworkDocument",
  MsgIncreaseActiveGovernanceFrameworkVersion: "/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
  MsgAddDID: "/verana.dd.v1.MsgAddDID",
  MsgRenewDID: "/verana.dd.v1.MsgRenewDID",
  MsgRemoveDID: "/verana.dd.v1.MsgRemoveDID",
  MsgTouchDID: "/verana.dd.v1.MsgTouchDID",
  MsgCreateCredentialSchema: "/verana.cs.v1.MsgCreateCredentialSchema",
  MsgUpdateCredentialSchema: "/verana.cs.v1.MsgUpdateCredentialSchema",
  MsgArchiveCredentialSchema: "/verana.cs.v1.MsgArchiveCredentialSchema",
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
  MsgReclaimTrustDeposit: "/verana.td.v1.MsgReclaimTrustDeposit",
  MsgReclaimTrustDepositYield: "/verana.td.v1.MsgReclaimTrustDepositYield",
  MsgSlashTrustDeposit: "/verana.td.v1.MsgSlashTrustDeposit",
  MsgRepaySlashedTrustDeposit: "/verana.td.v1.MsgRepaySlashedTrustDeposit",
} as const;

export const veranaRegistryTypes: ReadonlyArray<[string, GeneratedType]> = [
  [veranaTypeUrls.MsgCreateTrustRegistry, MsgCreateTrustRegistry as GeneratedType],
  [veranaTypeUrls.MsgUpdateTrustRegistry, MsgUpdateTrustRegistry as GeneratedType],
  [veranaTypeUrls.MsgArchiveTrustRegistry, MsgArchiveTrustRegistry as GeneratedType],
  [veranaTypeUrls.MsgAddGovernanceFrameworkDocument, MsgAddGovernanceFrameworkDocument as GeneratedType],
  [veranaTypeUrls.MsgIncreaseActiveGovernanceFrameworkVersion, MsgIncreaseActiveGovernanceFrameworkVersion as GeneratedType],
  [veranaTypeUrls.MsgAddDID, MsgAddDID as GeneratedType],
  [veranaTypeUrls.MsgRenewDID, MsgRenewDID as GeneratedType],
  [veranaTypeUrls.MsgRemoveDID, MsgRemoveDID as GeneratedType],
  [veranaTypeUrls.MsgTouchDID, MsgTouchDID as GeneratedType],
  [veranaTypeUrls.MsgCreateCredentialSchema, MsgCreateCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgUpdateCredentialSchema, MsgUpdateCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgArchiveCredentialSchema, MsgArchiveCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgCreatePermission, MsgCreatePermission as GeneratedType],
  [veranaTypeUrls.MsgCreateRootPermission, MsgCreateRootPermission as GeneratedType],
  [veranaTypeUrls.MsgExtendPermission, MsgExtendPermission as GeneratedType],
  [veranaTypeUrls.MsgRevokePermission, MsgRevokePermission as GeneratedType],
  [veranaTypeUrls.MsgStartPermissionVP, MsgStartPermissionVP as GeneratedType],
  [veranaTypeUrls.MsgRenewPermissionVP, MsgRenewPermissionVP as GeneratedType],
  [veranaTypeUrls.MsgSetPermissionVPToValidated, MsgSetPermissionVPToValidated as GeneratedType],
  [veranaTypeUrls.MsgCancelPermissionVPLastRequest, MsgCancelPermissionVPLastRequest as GeneratedType],
  [veranaTypeUrls.MsgCreateOrUpdatePermissionSession, MsgCreateOrUpdatePermissionSession as GeneratedType],
  [veranaTypeUrls.MsgSlashPermissionTrustDeposit, MsgSlashPermissionTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgRepayPermissionSlashedTrustDeposit, MsgRepayPermissionSlashedTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgReclaimTrustDeposit, MsgReclaimTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgReclaimTrustDepositYield, MsgReclaimTrustDepositYield as GeneratedType],
  [veranaTypeUrls.MsgSlashTrustDeposit, MsgSlashTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgRepaySlashedTrustDeposit, MsgRepaySlashedTrustDeposit as GeneratedType],
];

export function createVeranaRegistry(): Registry {
  const registry = new Registry(defaultRegistryTypes);
  for (const [typeUrl, generatedType] of veranaRegistryTypes) {
    registry.register(typeUrl, generatedType);
  }
  return registry;
}

export function createVeranaAminoTypes(): AminoTypes {
  return new AminoTypes({
    [veranaTypeUrls.MsgCreateTrustRegistry]: MsgCreateTrustRegistryAminoConverter,
    [veranaTypeUrls.MsgUpdateTrustRegistry]: MsgUpdateTrustRegistryAminoConverter,
    [veranaTypeUrls.MsgArchiveTrustRegistry]: MsgArchiveTrustRegistryAminoConverter,
    [veranaTypeUrls.MsgAddGovernanceFrameworkDocument]: MsgAddGovernanceFrameworkDocumentAminoConverter,
    [veranaTypeUrls.MsgIncreaseActiveGovernanceFrameworkVersion]: MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
    [veranaTypeUrls.MsgAddDID]: MsgAddDIDAminoConverter,
    [veranaTypeUrls.MsgRenewDID]: MsgRenewDIDAminoConverter,
    [veranaTypeUrls.MsgRemoveDID]: MsgRemoveDIDAminoConverter,
    [veranaTypeUrls.MsgTouchDID]: MsgTouchDIDAminoConverter,
    [veranaTypeUrls.MsgCreateCredentialSchema]: MsgCreateCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgUpdateCredentialSchema]: MsgUpdateCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgArchiveCredentialSchema]: MsgArchiveCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgCreatePermission]: MsgCreatePermissionAminoConverter,
    [veranaTypeUrls.MsgCreateRootPermission]: MsgCreateRootPermissionAminoConverter,
    [veranaTypeUrls.MsgExtendPermission]: MsgExtendPermissionAminoConverter,
    [veranaTypeUrls.MsgRevokePermission]: MsgRevokePermissionAminoConverter,
    [veranaTypeUrls.MsgStartPermissionVP]: MsgStartPermissionVPAminoConverter,
    [veranaTypeUrls.MsgRenewPermissionVP]: MsgRenewPermissionVPAminoConverter,
    [veranaTypeUrls.MsgSetPermissionVPToValidated]: MsgSetPermissionVPToValidatedAminoConverter,
    [veranaTypeUrls.MsgCancelPermissionVPLastRequest]: MsgCancelPermissionVPLastRequestAminoConverter,
    [veranaTypeUrls.MsgCreateOrUpdatePermissionSession]: MsgCreateOrUpdatePermissionSessionAminoConverter,
    [veranaTypeUrls.MsgSlashPermissionTrustDeposit]: MsgSlashPermissionTrustDepositAminoConverter,
    [veranaTypeUrls.MsgRepayPermissionSlashedTrustDeposit]: MsgRepayPermissionSlashedTrustDepositAminoConverter,
    [veranaTypeUrls.MsgReclaimTrustDeposit]: MsgReclaimTrustDepositAminoConverter,
    [veranaTypeUrls.MsgReclaimTrustDepositYield]: MsgReclaimTrustDepositYieldAminoConverter,
    [veranaTypeUrls.MsgRepaySlashedTrustDeposit]: MsgRepaySlashedTrustDepositAminoConverter,
  });
}
