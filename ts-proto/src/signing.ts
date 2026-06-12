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
  MsgSetParticipantEffectiveUntil,
  MsgCancelParticipantOPLastRequest,
  MsgCreateOrUpdateParticipantSession,
  MsgSelfCreateParticipant,
  MsgCreateRootParticipant,
  MsgRenewParticipantOP,
  MsgRepayParticipantSlashedTrustDeposit,
  MsgRevokeParticipant,
  MsgSetParticipantOPToValidated,
  MsgSlashParticipantTrustDeposit,
  MsgStartParticipantOP,
  MsgTriggerResolver,
} from "./codec/verana/pp/v1/tx";
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
  MsgSetParticipantEffectiveUntilAminoConverter,
  MsgCancelParticipantOPLastRequestAminoConverter,
  MsgCreateOrUpdateParticipantSessionAminoConverter,
  MsgSelfCreateParticipantAminoConverter,
  MsgCreateRootParticipantAminoConverter,
  MsgRenewParticipantOPAminoConverter,
  MsgRepayParticipantSlashedTrustDepositAminoConverter,
  MsgRevokeParticipantAminoConverter,
  MsgSetParticipantOPToValidatedAminoConverter,
  MsgSlashParticipantTrustDepositAminoConverter,
  MsgStartParticipantOPAminoConverter,
  MsgTriggerResolverAminoConverter,
} from "./amino-converter/pp";
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
  MsgSelfCreateParticipant: "/verana.pp.v1.MsgSelfCreateParticipant",
  MsgCreateRootParticipant: "/verana.pp.v1.MsgCreateRootParticipant",
  MsgSetParticipantEffectiveUntil: "/verana.pp.v1.MsgSetParticipantEffectiveUntil",
  MsgRevokeParticipant: "/verana.pp.v1.MsgRevokeParticipant",
  MsgStartParticipantOP: "/verana.pp.v1.MsgStartParticipantOP",
  MsgRenewParticipantOP: "/verana.pp.v1.MsgRenewParticipantOP",
  MsgSetParticipantOPToValidated: "/verana.pp.v1.MsgSetParticipantOPToValidated",
  MsgTriggerResolver: "/verana.pp.v1.MsgTriggerResolver",
  MsgCancelParticipantOPLastRequest: "/verana.pp.v1.MsgCancelParticipantOPLastRequest",
  MsgCreateOrUpdateParticipantSession: "/verana.pp.v1.MsgCreateOrUpdateParticipantSession",
  MsgSlashParticipantTrustDeposit: "/verana.pp.v1.MsgSlashParticipantTrustDeposit",
  MsgRepayParticipantSlashedTrustDeposit: "/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit",
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
  [veranaTypeUrls.MsgSelfCreateParticipant, MsgSelfCreateParticipant as GeneratedType],
  [veranaTypeUrls.MsgCreateRootParticipant, MsgCreateRootParticipant as GeneratedType],
  [veranaTypeUrls.MsgSetParticipantEffectiveUntil, MsgSetParticipantEffectiveUntil as GeneratedType],
  [veranaTypeUrls.MsgRevokeParticipant, MsgRevokeParticipant as GeneratedType],
  [veranaTypeUrls.MsgStartParticipantOP, MsgStartParticipantOP as GeneratedType],
  [veranaTypeUrls.MsgRenewParticipantOP, MsgRenewParticipantOP as GeneratedType],
  [veranaTypeUrls.MsgSetParticipantOPToValidated, MsgSetParticipantOPToValidated as GeneratedType],
  [veranaTypeUrls.MsgTriggerResolver, MsgTriggerResolver as GeneratedType],
  [veranaTypeUrls.MsgCancelParticipantOPLastRequest, MsgCancelParticipantOPLastRequest as GeneratedType],
  [veranaTypeUrls.MsgCreateOrUpdateParticipantSession, MsgCreateOrUpdateParticipantSession as GeneratedType],
  [veranaTypeUrls.MsgSlashParticipantTrustDeposit, MsgSlashParticipantTrustDeposit as GeneratedType],
  [veranaTypeUrls.MsgRepayParticipantSlashedTrustDeposit, MsgRepayParticipantSlashedTrustDeposit as GeneratedType],
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
    [veranaTypeUrls.MsgSelfCreateParticipant]: MsgSelfCreateParticipantAminoConverter,
    [veranaTypeUrls.MsgCreateRootParticipant]: MsgCreateRootParticipantAminoConverter,
    [veranaTypeUrls.MsgSetParticipantEffectiveUntil]: MsgSetParticipantEffectiveUntilAminoConverter,
    [veranaTypeUrls.MsgRevokeParticipant]: MsgRevokeParticipantAminoConverter,
    [veranaTypeUrls.MsgStartParticipantOP]: MsgStartParticipantOPAminoConverter,
    [veranaTypeUrls.MsgRenewParticipantOP]: MsgRenewParticipantOPAminoConverter,
    [veranaTypeUrls.MsgSetParticipantOPToValidated]: MsgSetParticipantOPToValidatedAminoConverter,
    [veranaTypeUrls.MsgTriggerResolver]: MsgTriggerResolverAminoConverter,
    [veranaTypeUrls.MsgCancelParticipantOPLastRequest]: MsgCancelParticipantOPLastRequestAminoConverter,
    [veranaTypeUrls.MsgCreateOrUpdateParticipantSession]: MsgCreateOrUpdateParticipantSessionAminoConverter,
    [veranaTypeUrls.MsgSlashParticipantTrustDeposit]: MsgSlashParticipantTrustDepositAminoConverter,
    [veranaTypeUrls.MsgRepayParticipantSlashedTrustDeposit]: MsgRepayParticipantSlashedTrustDepositAminoConverter,
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
