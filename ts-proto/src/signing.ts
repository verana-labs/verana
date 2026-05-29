import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { AminoTypes, defaultRegistryTypes, createDefaultAminoConverters } from "@cosmjs/stargate";
import { createGroupAminoConverters } from "./amino-converter/group";
import {
  MsgCreateCorporation,
  MsgUpdateCorporation,
} from "./codec/verana/co/v1/tx";
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
  MsgArchiveEcosystem,
  MsgCreateEcosystem,
  MsgUpdateEcosystem,
} from "./codec/verana/ec/v1/tx";
import {
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "./codec/verana/gf/v1/tx";
import {
  MsgAdjustPermission,
  MsgCancelPermissionVPLastRequest,
  MsgCreateOrUpdatePermissionSession,
  MsgSelfCreatePermission,
  MsgCreateRootPermission,
  MsgRenewPermissionVP,
  MsgRepayPermissionSlashedTrustDeposit,
  MsgRevokePermission,
  MsgSetPermissionVPToValidated,
  MsgSlashPermissionTrustDeposit,
  MsgStartPermissionVP,
} from "./codec/verana/perm/v1/tx";
import {
  MsgReclaimTrustDepositYield,
  MsgRepaySlashedTrustDeposit,
  MsgSlashTrustDeposit,
} from "./codec/verana/td/v1/tx";
import {
  MsgCreateExchangeRate,
  MsgSetExchangeRateState,
  MsgUpdateExchangeRate,
} from "./codec/verana/xr/v1/tx";
import {
  MsgCreateCorporationAminoConverter,
  MsgUpdateCorporationAminoConverter,
} from "./amino-converter/co";
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
  MsgArchiveEcosystemAminoConverter,
  MsgCreateEcosystemAminoConverter,
  MsgUpdateEcosystemAminoConverter,
} from "./amino-converter/ec";
import {
  MsgAddGovernanceFrameworkDocumentAminoConverter,
  MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
} from "./amino-converter/gf";
import {
  MsgAdjustPermissionAminoConverter,
  MsgCancelPermissionVPLastRequestAminoConverter,
  MsgCreateOrUpdatePermissionSessionAminoConverter,
  MsgSelfCreatePermissionAminoConverter,
  MsgCreateRootPermissionAminoConverter,
  MsgRenewPermissionVPAminoConverter,
  MsgRepayPermissionSlashedTrustDepositAminoConverter,
  MsgRevokePermissionAminoConverter,
  MsgSetPermissionVPToValidatedAminoConverter,
  MsgSlashPermissionTrustDepositAminoConverter,
  MsgStartPermissionVPAminoConverter,
} from "./amino-converter/perm";
import {
  MsgReclaimTrustDepositYieldAminoConverter,
  MsgRepaySlashedTrustDepositAminoConverter,
  MsgSlashTrustDepositAminoConverter,
} from "./amino-converter/td";
import {
  MsgCreateExchangeRateAminoConverter,
  MsgSetExchangeRateStateAminoConverter,
  MsgUpdateExchangeRateAminoConverter,
} from "./amino-converter/xr";

export const veranaTypeUrls = {
  MsgCreateCorporation: "/verana.co.v1.MsgCreateCorporation",
  MsgUpdateCorporation: "/verana.co.v1.MsgUpdateCorporation",
  MsgCreateEcosystem: "/verana.ec.v1.MsgCreateEcosystem",
  MsgUpdateEcosystem: "/verana.ec.v1.MsgUpdateEcosystem",
  MsgArchiveEcosystem: "/verana.ec.v1.MsgArchiveEcosystem",
  MsgAddGovernanceFrameworkDocument: "/verana.gf.v1.MsgAddGovernanceFrameworkDocument",
  MsgIncreaseActiveGovernanceFrameworkVersion: "/verana.gf.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
  MsgCreateCredentialSchema: "/verana.cs.v1.MsgCreateCredentialSchema",
  MsgUpdateCredentialSchema: "/verana.cs.v1.MsgUpdateCredentialSchema",
  MsgArchiveCredentialSchema: "/verana.cs.v1.MsgArchiveCredentialSchema",
  MsgSelfCreatePermission: "/verana.perm.v1.MsgSelfCreatePermission",
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
  MsgReclaimTrustDepositYield: "/verana.td.v1.MsgReclaimTrustDepositYield",
  MsgSlashTrustDeposit: "/verana.td.v1.MsgSlashTrustDeposit",
  MsgRepaySlashedTrustDeposit: "/verana.td.v1.MsgRepaySlashedTrustDeposit",
  MsgGrantOperatorAuthorization: "/verana.de.v1.MsgGrantOperatorAuthorization",
  MsgRevokeOperatorAuthorization: "/verana.de.v1.MsgRevokeOperatorAuthorization",
  MsgStoreDigest: "/verana.di.v1.MsgStoreDigest",
  MsgCreateExchangeRate: "/verana.xr.v1.MsgCreateExchangeRate",
  MsgUpdateExchangeRate: "/verana.xr.v1.MsgUpdateExchangeRate",
  MsgSetExchangeRateState: "/verana.xr.v1.MsgSetExchangeRateState",
} as const;

export const typeUrls = veranaTypeUrls;

export const veranaRegistryTypes: ReadonlyArray<[string, GeneratedType]> = [
  [veranaTypeUrls.MsgCreateCorporation, MsgCreateCorporation as GeneratedType],
  [veranaTypeUrls.MsgUpdateCorporation, MsgUpdateCorporation as GeneratedType],
  [veranaTypeUrls.MsgCreateEcosystem, MsgCreateEcosystem as GeneratedType],
  [veranaTypeUrls.MsgUpdateEcosystem, MsgUpdateEcosystem as GeneratedType],
  [veranaTypeUrls.MsgArchiveEcosystem, MsgArchiveEcosystem as GeneratedType],
  [veranaTypeUrls.MsgAddGovernanceFrameworkDocument, MsgAddGovernanceFrameworkDocument as GeneratedType],
  [veranaTypeUrls.MsgIncreaseActiveGovernanceFrameworkVersion, MsgIncreaseActiveGovernanceFrameworkVersion as GeneratedType],
  [veranaTypeUrls.MsgCreateCredentialSchema, MsgCreateCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgUpdateCredentialSchema, MsgUpdateCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgArchiveCredentialSchema, MsgArchiveCredentialSchema as GeneratedType],
  [veranaTypeUrls.MsgSelfCreatePermission, MsgSelfCreatePermission as GeneratedType],
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
  [veranaTypeUrls.MsgReclaimTrustDepositYield, MsgReclaimTrustDepositYield as GeneratedType],
  [veranaTypeUrls.MsgSlashTrustDeposit, MsgSlashTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgRepaySlashedTrustDeposit, MsgRepaySlashedTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgGrantOperatorAuthorization, MsgGrantOperatorAuthorization as GeneratedType],
  [veranaTypeUrls.MsgRevokeOperatorAuthorization, MsgRevokeOperatorAuthorization as GeneratedType],
  [veranaTypeUrls.MsgStoreDigest, MsgStoreDigest as GeneratedType],
  [veranaTypeUrls.MsgCreateExchangeRate, MsgCreateExchangeRate as GeneratedType],
  [veranaTypeUrls.MsgUpdateExchangeRate, MsgUpdateExchangeRate as GeneratedType],
  [veranaTypeUrls.MsgSetExchangeRateState, MsgSetExchangeRateState as GeneratedType],
];

export function createVeranaRegistry(): Registry {
  const registry = new Registry(defaultRegistryTypes);
  for (const [typeUrl, generatedType] of veranaRegistryTypes) {
    registry.register(typeUrl, generatedType);
  }
  return registry;
}

export function createVeranaAminoTypes(): AminoTypes {
  const registry = createVeranaRegistry();
  let aminoTypesRef: AminoTypes;
  const groupConverters = createGroupAminoConverters(() => aminoTypesRef, registry);
  aminoTypesRef = new AminoTypes({
    ...createDefaultAminoConverters(),
    ...groupConverters,
    [veranaTypeUrls.MsgCreateCorporation]: MsgCreateCorporationAminoConverter,
    [veranaTypeUrls.MsgUpdateCorporation]: MsgUpdateCorporationAminoConverter,
    [veranaTypeUrls.MsgCreateEcosystem]: MsgCreateEcosystemAminoConverter,
    [veranaTypeUrls.MsgUpdateEcosystem]: MsgUpdateEcosystemAminoConverter,
    [veranaTypeUrls.MsgArchiveEcosystem]: MsgArchiveEcosystemAminoConverter,
    [veranaTypeUrls.MsgAddGovernanceFrameworkDocument]: MsgAddGovernanceFrameworkDocumentAminoConverter,
    [veranaTypeUrls.MsgIncreaseActiveGovernanceFrameworkVersion]: MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
    [veranaTypeUrls.MsgCreateCredentialSchema]: MsgCreateCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgUpdateCredentialSchema]: MsgUpdateCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgArchiveCredentialSchema]: MsgArchiveCredentialSchemaAminoConverter,
    [veranaTypeUrls.MsgSelfCreatePermission]: MsgSelfCreatePermissionAminoConverter,
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
    [veranaTypeUrls.MsgReclaimTrustDepositYield]: MsgReclaimTrustDepositYieldAminoConverter,
    [veranaTypeUrls.MsgSlashTrustDeposit]: MsgSlashTrustDepositAminoConverter,
    [veranaTypeUrls.MsgRepaySlashedTrustDeposit]: MsgRepaySlashedTrustDepositAminoConverter,
    [veranaTypeUrls.MsgGrantOperatorAuthorization]: MsgGrantOperatorAuthorizationAminoConverter,
    [veranaTypeUrls.MsgRevokeOperatorAuthorization]: MsgRevokeOperatorAuthorizationAminoConverter,
    [veranaTypeUrls.MsgStoreDigest]: MsgStoreDigestAminoConverter,
    [veranaTypeUrls.MsgCreateExchangeRate]: MsgCreateExchangeRateAminoConverter,
    [veranaTypeUrls.MsgUpdateExchangeRate]: MsgUpdateExchangeRateAminoConverter,
    [veranaTypeUrls.MsgSetExchangeRateState]: MsgSetExchangeRateStateAminoConverter,
  });
  return aminoTypesRef;
}
