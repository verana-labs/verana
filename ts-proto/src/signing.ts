import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { AminoTypes, defaultRegistryTypes } from "@cosmjs/stargate";
import {
  MsgArchiveCredentialSchema,
  MsgCreateCredentialSchema,
  MsgUpdateCredentialSchema,
} from "./codec/verana/cs/v1/tx";
import {
  MsgGrantOperatorAuthorization,
  MsgRevokeOperatorAuthorization,
} from "./codec/verana/de/v1/tx";
import { MsgStoreDigest } from "./codec/verana/di/v1/tx";
import {
  MsgAdjustPermission,
  MsgCancelPermissionVPLastRequest,
  MsgCreateOrUpdatePermissionSession,
  MsgCreatePermission,
  MsgCreateRootPermission,
  MsgRenewPermissionVP,
  MsgRepayPermissionSlashedTrustDeposit,
  MsgRevokePermission,
  MsgSetPermissionVPToValidated,
  MsgSlashPermissionTrustDeposit,
  MsgStartPermissionVP,
} from "./codec/verana/perm/v1/tx";
import {
  MsgReclaimTrustDeposit,
  MsgReclaimTrustDepositYield,
  MsgRepaySlashedTrustDeposit,
  MsgSlashTrustDeposit,
} from "./codec/verana/td/v1/tx";
import {
  MsgAddGovernanceFrameworkDocument,
  MsgArchiveTrustRegistry,
  MsgCreateTrustRegistry,
  MsgIncreaseActiveGovernanceFrameworkVersion,
  MsgUpdateTrustRegistry,
} from "./codec/verana/tr/v1/tx";
import {
  MsgCreateExchangeRate,
  MsgToggleExchangeRateState,
  MsgUpdateExchangeRate,
} from "./codec/verana/xr/v1/tx";
import {
  MsgArchiveCredentialSchemaAminoConverter,
  MsgCreateCredentialSchemaAminoConverter,
  MsgUpdateCredentialSchemaAminoConverter,
} from "./amino-converter/cs";
import {
  MsgGrantOperatorAuthorizationAminoConverter,
  MsgRevokeOperatorAuthorizationAminoConverter,
} from "./amino-converter/de";
import { MsgStoreDigestAminoConverter } from "./amino-converter/di";
import {
  MsgAdjustPermissionAminoConverter,
  MsgCancelPermissionVPLastRequestAminoConverter,
  MsgCreateOrUpdatePermissionSessionAminoConverter,
  MsgCreatePermissionAminoConverter,
  MsgCreateRootPermissionAminoConverter,
  MsgRenewPermissionVPAminoConverter,
  MsgRepayPermissionSlashedTrustDepositAminoConverter,
  MsgRevokePermissionAminoConverter,
  MsgSetPermissionVPToValidatedAminoConverter,
  MsgSlashPermissionTrustDepositAminoConverter,
  MsgStartPermissionVPAminoConverter,
} from "./amino-converter/perm";
import {
  MsgReclaimTrustDepositAminoConverter,
  MsgReclaimTrustDepositYieldAminoConverter,
  MsgRepaySlashedTrustDepositAminoConverter,
  MsgSlashTrustDepositAminoConverter,
} from "./amino-converter/td";
import {
  MsgAddGovernanceFrameworkDocumentAminoConverter,
  MsgArchiveTrustRegistryAminoConverter,
  MsgCreateTrustRegistryAminoConverter,
  MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
  MsgUpdateTrustRegistryAminoConverter,
} from "./amino-converter/tr";
import {
  MsgCreateExchangeRateAminoConverter,
  MsgToggleExchangeRateStateAminoConverter,
  MsgUpdateExchangeRateAminoConverter,
} from "./amino-converter/xr";

export const veranaTypeUrls = {
  MsgCreateTrustRegistry: "/verana.tr.v1.MsgCreateTrustRegistry",
  MsgUpdateTrustRegistry: "/verana.tr.v1.MsgUpdateTrustRegistry",
  MsgArchiveTrustRegistry: "/verana.tr.v1.MsgArchiveTrustRegistry",
  MsgAddGovernanceFrameworkDocument: "/verana.tr.v1.MsgAddGovernanceFrameworkDocument",
  MsgIncreaseActiveGovernanceFrameworkVersion: "/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
  MsgCreateCredentialSchema: "/verana.cs.v1.MsgCreateCredentialSchema",
  MsgUpdateCredentialSchema: "/verana.cs.v1.MsgUpdateCredentialSchema",
  MsgArchiveCredentialSchema: "/verana.cs.v1.MsgArchiveCredentialSchema",
  MsgCreatePermission: "/verana.perm.v1.MsgCreatePermission",
  MsgCreateRootPermission: "/verana.perm.v1.MsgCreateRootPermission",
  MsgAdjustPermission: "/verana.perm.v1.MsgAdjustPermission",
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
  MsgGrantOperatorAuthorization: "/verana.de.v1.MsgGrantOperatorAuthorization",
  MsgRevokeOperatorAuthorization: "/verana.de.v1.MsgRevokeOperatorAuthorization",
  MsgStoreDigest: "/verana.di.v1.MsgStoreDigest",
  MsgCreateExchangeRate: "/verana.xr.v1.MsgCreateExchangeRate",
  MsgUpdateExchangeRate: "/verana.xr.v1.MsgUpdateExchangeRate",
  MsgToggleExchangeRateState: "/verana.xr.v1.MsgToggleExchangeRateState",
} as const;

export const typeUrls = veranaTypeUrls;

export const veranaRegistryTypes: ReadonlyArray<[string, GeneratedType]> = [
  [veranaTypeUrls.MsgCreateTrustRegistry, MsgCreateTrustRegistry as GeneratedType],
  [veranaTypeUrls.MsgUpdateTrustRegistry, MsgUpdateTrustRegistry as GeneratedType],
  [veranaTypeUrls.MsgArchiveTrustRegistry, MsgArchiveTrustRegistry as GeneratedType],
  [veranaTypeUrls.MsgAddGovernanceFrameworkDocument, MsgAddGovernanceFrameworkDocument as GeneratedType],
  [veranaTypeUrls.MsgIncreaseActiveGovernanceFrameworkVersion, MsgIncreaseActiveGovernanceFrameworkVersion as GeneratedType],
  [veranaTypeUrls.MsgCreateCredentialSchema, MsgCreateCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgUpdateCredentialSchema, MsgUpdateCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgArchiveCredentialSchema, MsgArchiveCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgCreatePermission, MsgCreatePermission as GeneratedType],
  [veranaTypeUrls.MsgCreateRootPermission, MsgCreateRootPermission as GeneratedType],
  [veranaTypeUrls.MsgAdjustPermission, MsgAdjustPermission as GeneratedType],
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
  [veranaTypeUrls.MsgGrantOperatorAuthorization, MsgGrantOperatorAuthorization as GeneratedType],
  [veranaTypeUrls.MsgRevokeOperatorAuthorization, MsgRevokeOperatorAuthorization as GeneratedType],
  [veranaTypeUrls.MsgStoreDigest, MsgStoreDigest as GeneratedType],
  [veranaTypeUrls.MsgCreateExchangeRate, MsgCreateExchangeRate as GeneratedType],
  [veranaTypeUrls.MsgUpdateExchangeRate, MsgUpdateExchangeRate as GeneratedType],
  [veranaTypeUrls.MsgToggleExchangeRateState, MsgToggleExchangeRateState as GeneratedType],
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
    [veranaTypeUrls.MsgCreateCredentialSchema]: MsgCreateCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgUpdateCredentialSchema]: MsgUpdateCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgArchiveCredentialSchema]: MsgArchiveCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgCreatePermission]: MsgCreatePermissionAminoConverter,
    [veranaTypeUrls.MsgCreateRootPermission]: MsgCreateRootPermissionAminoConverter,
    [veranaTypeUrls.MsgAdjustPermission]: MsgAdjustPermissionAminoConverter,
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
    [veranaTypeUrls.MsgSlashTrustDeposit]: MsgSlashTrustDepositAminoConverter,
    [veranaTypeUrls.MsgRepaySlashedTrustDeposit]: MsgRepaySlashedTrustDepositAminoConverter,
    [veranaTypeUrls.MsgGrantOperatorAuthorization]: MsgGrantOperatorAuthorizationAminoConverter,
    [veranaTypeUrls.MsgRevokeOperatorAuthorization]: MsgRevokeOperatorAuthorizationAminoConverter,
    [veranaTypeUrls.MsgStoreDigest]: MsgStoreDigestAminoConverter,
    [veranaTypeUrls.MsgCreateExchangeRate]: MsgCreateExchangeRateAminoConverter,
    [veranaTypeUrls.MsgUpdateExchangeRate]: MsgUpdateExchangeRateAminoConverter,
    [veranaTypeUrls.MsgToggleExchangeRateState]: MsgToggleExchangeRateStateAminoConverter,
  });
}
