/**
 * Amino Converters for Verana Message Types
 * 
 * These converters match the frontend implementation to ensure compatibility.
 * They convert between Proto messages and Amino JSON format.
 */

import type { AminoConverter } from "@cosmjs/stargate";
import Long from "long";
import {
  MsgCreateTrustRegistry,
  MsgUpdateTrustRegistry,
  MsgArchiveTrustRegistry,
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "../../../src/codec/verana/tr/v1/tx";
import {
  MsgAddDID,
  MsgRenewDID,
  MsgTouchDID,
  MsgRemoveDID,
} from "../../../src/codec/verana/dd/v1/tx";
import {
  MsgCreateCredentialSchema,
  MsgUpdateCredentialSchema,
  MsgArchiveCredentialSchema,
} from "../../../src/codec/verana/cs/v1/tx";
import {
  MsgReclaimTrustDeposit,
  MsgReclaimTrustDepositYield,
} from "../../../src/codec/verana/td/v1/tx";
import {
  MsgCreateRootPermission,
  MsgCreatePermission,
  MsgExtendPermission,
  MsgRevokePermission,
  MsgStartPermissionVP,
  MsgRenewPermissionVP,
  MsgSetPermissionVPToValidated,
  MsgCancelPermissionVPLastRequest,
  MsgCreateOrUpdatePermissionSession,
} from "../../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../../src/codec/verana/perm/v1/types";

// Helper functions for Amino conversion (matching frontend)
const u64ToStr = (v?: Long | string | number | null) =>
  v != null ? Long.fromValue(v).toString() : undefined;

const u64ToStrIfNonZero = (v?: Long | string | number | null) => {
  if (v == null) return undefined;
  const value = Long.fromValue(v);
  return value.isZero() ? undefined : value.toString();
};

const strToU64 = (s?: string | null) =>
  s != null ? Long.fromString(s) : undefined;

const u32ToAmino = (n?: number | null) =>
  n == null ? undefined : (((n >>> 0) === 0) ? 0 : (n >>> 0));

const pickU32 = (v?: number | string | null) =>
  v == null ? undefined : (Number(v) >>> 0);

// Helper for OptionalUInt32: 0 => {} (omitempty), >0 => {value:n}
const toOptU32Amino = (m?: { value: number } | undefined) => {
  if (!m) return undefined;
  const value = (Number(m.value) >>> 0);
  return value === 0 ? {} : { value };
};

// Helper for OptionalUInt32: {} (=> 0), {value:n} => OptionalUInt32
const fromOptU32Amino = (x: any): { value: number } | undefined => {
  if (x == null) return undefined;
  // {} => wrapper, value default 0
  if (typeof x === "object" && x.value == null) return { value: 0 };
  const n = typeof x === "object" ? x.value : x;
  if (n === undefined || n === null) return undefined;
  if (typeof n === "string" && n.trim() === "") return undefined;
  const u = (Number(n) >>> 0);
  return { value: u };
};

const clean = <T extends Record<string, any>>(o: T): T => {
  Object.keys(o).forEach((k) => o[k] === undefined && delete o[k]);
  return o;
};

// Helper for Date/Timestamp: Date -> ISO string, ISO string -> Date
const dateToAmino = (d?: Date | null) =>
  d != null ? d.toISOString() : undefined;

const dateFromAmino = (s?: string | null) =>
  s != null ? new Date(s) : undefined;

// ============================================================================
// Trust Registry (TR) Module Converters
// ============================================================================

export const MsgCreateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: '/verana.tr.v1.MsgCreateTrustRegistry',
  toAmino: ({
    creator,
    did,
    aka,
    language,
    docUrl,
    docDigestSri,
  }: MsgCreateTrustRegistry) => ({
    creator,
    did,
    aka,
    language,
    doc_url: docUrl,
    doc_digest_sri: docDigestSri,
  }),
  fromAmino: (value: {
    creator: string;
    did: string;
    aka: string;
    language: string;
    doc_url: string;
    doc_digest_sri: string;
  }) =>
    MsgCreateTrustRegistry.fromPartial({
      creator: value.creator,
      did: value.did,
      aka: value.aka,
      language: value.language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
    }),
};

export const MsgUpdateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: '/verana.tr.v1.MsgUpdateTrustRegistry',
  toAmino: ({ creator, id, did, aka }: MsgUpdateTrustRegistry) => ({
    creator,
    id: id != null ? id.toString() : undefined,
    did,
    aka,
  }),
  fromAmino: (value: { creator: string; id: string; did: string; aka: string }) =>
    MsgUpdateTrustRegistry.fromPartial({
      creator: value.creator,
      id: value.id != null ? Number(value.id) : 0,
      did: value.did,
      aka: value.aka,
    }),
};

export const MsgArchiveTrustRegistryAminoConverter: AminoConverter = {
  aminoType: '/verana.tr.v1.MsgArchiveTrustRegistry',
  toAmino: ({ creator, id, archive }: MsgArchiveTrustRegistry) => ({
    creator,
    id: id != null ? id.toString() : undefined,
    archive,
  }),
  fromAmino: (value: { creator: string; id: string; archive: boolean }) =>
    MsgArchiveTrustRegistry.fromPartial({
      creator: value.creator,
      id: value.id != null ? Number(value.id) : 0,
      archive: value.archive,
    }),
};

export const MsgAddGovernanceFrameworkDocumentAminoConverter: AminoConverter = {
  aminoType: '/verana.tr.v1.MsgAddGovernanceFrameworkDocument',
  toAmino: ({
    creator,
    id,
    docLanguage,
    docUrl,
    docDigestSri,
    version,
  }: MsgAddGovernanceFrameworkDocument) => ({
    creator,
    id: id != null ? id.toString() : undefined,
    doc_language: docLanguage,
    doc_url: docUrl,
    doc_digest_sri: docDigestSri,
    version,
  }),
  fromAmino: (value: {
    creator: string;
    id: string;
    doc_language: string;
    doc_url: string;
    doc_digest_sri: string;
    version: number;
  }) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      creator: value.creator,
      id: value.id != null ? Number(value.id) : 0,
      docLanguage: value.doc_language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
      version: value.version,
    }),
};

export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: '/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion',
  toAmino: ({ creator, id }: MsgIncreaseActiveGovernanceFrameworkVersion) => ({
    creator,
    id: id != null ? id.toString() : undefined,
  }),
  fromAmino: (value: { creator: string; id: string }) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      creator: value.creator,
      id: value.id != null ? Number(value.id) : 0,
    }),
};

// ============================================================================
// DID Directory (DD) Module Converters
// ============================================================================

export const MsgAddDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgAddDID',
  toAmino: ({ creator, did, years }: MsgAddDID) => ({
    creator,
    did,
    years,
  }),
  fromAmino: (value: { creator: string; did: string; years: number }) =>
    MsgAddDID.fromPartial({
      creator: value.creator,
      did: value.did,
      years: value.years,
    }),
};

export const MsgRenewDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgRenewDID',
  toAmino: ({ creator, did, years }: MsgRenewDID) => ({
    creator,
    did,
    years,
  }),
  fromAmino: (value: { creator: string; did: string; years: number }) =>
    MsgRenewDID.fromPartial({
      creator: value.creator,
      did: value.did,
      years: value.years,
    }),
};

export const MsgTouchDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgTouchDID',
  toAmino: ({ creator, did }: MsgTouchDID) => ({
    creator,
    did,
  }),
  fromAmino: (value: { creator: string; did: string }) =>
    MsgTouchDID.fromPartial({
      creator: value.creator,
      did: value.did,
    }),
};

export const MsgRemoveDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgRemoveDID',
  toAmino: ({ creator, did }: MsgRemoveDID) => ({
    creator,
    did,
  }),
  fromAmino: (value: { creator: string; did: string }) =>
    MsgRemoveDID.fromPartial({
      creator: value.creator,
      did: value.did,
    }),
};

// ============================================================================
// Credential Schema (CS) Module Converters
// ============================================================================

export const MsgCreateCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: '/verana.cs.v1.MsgCreateCredentialSchema',
  toAmino: (m: MsgCreateCredentialSchema) => clean({
    creator: m.creator ?? '',
    tr_id: u64ToStr(m.trId), // Use u64ToStr to match frontend (handles number/Long)
    json_schema: m.jsonSchema ?? '',
    issuer_grantor_validation_validity_period: toOptU32Amino(m.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(m.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(m.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(m.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(m.holderValidationValidityPeriod),
    issuer_perm_management_mode: u32ToAmino(m.issuerPermManagementMode),
    verifier_perm_management_mode: u32ToAmino(m.verifierPermManagementMode),
  }),
  fromAmino: (a: any): MsgCreateCredentialSchema =>
    MsgCreateCredentialSchema.fromPartial({
      creator: a.creator ?? '',
      trId: strToU64(a.tr_id) != null ? Number(strToU64(a.tr_id)!.toString()) : 0, // Convert Long to number
      jsonSchema: a.json_schema ?? '',
      issuerGrantorValidationValidityPeriod: fromOptU32Amino(a.issuer_grantor_validation_validity_period),
      verifierGrantorValidationValidityPeriod: fromOptU32Amino(a.verifier_grantor_validation_validity_period),
      issuerValidationValidityPeriod: fromOptU32Amino(a.issuer_validation_validity_period),
      verifierValidationValidityPeriod: fromOptU32Amino(a.verifier_validation_validity_period),
      holderValidationValidityPeriod: fromOptU32Amino(a.holder_validation_validity_period),
      issuerPermManagementMode: a.issuer_perm_management_mode ?? 0,
      verifierPermManagementMode: a.verifier_perm_management_mode ?? 0,
    }),
};

export const MsgUpdateCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: '/verana.cs.v1.MsgUpdateCredentialSchema',
  toAmino: (m: MsgUpdateCredentialSchema) => clean({
    creator: m.creator ?? '',
    id: u64ToStr(m.id), // Use u64ToStr to match frontend
    issuer_grantor_validation_validity_period: toOptU32Amino(m.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(m.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(m.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(m.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(m.holderValidationValidityPeriod),
  }),
  fromAmino: (a: any) => MsgUpdateCredentialSchema.fromPartial({
    creator: a.creator ?? '',
    id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0, // Convert Long to number
    issuerGrantorValidationValidityPeriod: fromOptU32Amino(a.issuer_grantor_validation_validity_period),
    verifierGrantorValidationValidityPeriod: fromOptU32Amino(a.verifier_grantor_validation_validity_period),
    issuerValidationValidityPeriod: fromOptU32Amino(a.issuer_validation_validity_period),
    verifierValidationValidityPeriod: fromOptU32Amino(a.verifier_validation_validity_period),
    holderValidationValidityPeriod: fromOptU32Amino(a.holder_validation_validity_period),
  }),
};

export const MsgArchiveCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: '/verana.cs.v1.MsgArchiveCredentialSchema',
  toAmino: (m: MsgArchiveCredentialSchema) => ({
    creator: m.creator ?? '',
    id: u64ToStr(m.id), // Use u64ToStr to match frontend
    archive: m.archive ?? false,
  }),
  fromAmino: (a: { creator: string; id: string; archive: boolean }): MsgArchiveCredentialSchema =>
    MsgArchiveCredentialSchema.fromPartial({
      creator: a.creator,
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0, // Convert Long to number
      archive: a.archive ?? false,
    }),
};

// ============================================================================
// Trust Deposit (TD) Module Converters
// ============================================================================

export const MsgReclaimTrustDepositAminoConverter: AminoConverter = {
  aminoType: '/verana.td.v1.MsgReclaimTrustDeposit',
  toAmino: ({ creator, claimed }: MsgReclaimTrustDeposit) => ({
    creator,
    claimed: claimed != null ? claimed.toString() : undefined,
  }),
  fromAmino: (value: { creator: string; claimed: number | string }) =>
    MsgReclaimTrustDeposit.fromPartial({
      creator: value.creator,
      claimed: value.claimed != null ? Number(value.claimed) : 0,
    }),
};

export const MsgReclaimTrustDepositYieldAminoConverter: AminoConverter = {
  aminoType: '/verana.td.v1.MsgReclaimTrustDepositYield',
  toAmino: ({ creator }: MsgReclaimTrustDepositYield) => ({
    creator,
  }),
  fromAmino: (value: { creator: string }) =>
    MsgReclaimTrustDepositYield.fromPartial({
      creator: value.creator,
    }),
};

// ============================================================================
// Permission (PERM) Module Converters
// ============================================================================

export const MsgCreateRootPermissionAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgCreateRootPermission',
  toAmino: (m: MsgCreateRootPermission) => clean({
    creator: m.creator ?? '',
    schema_id: u64ToStr(m.schemaId),
    did: m.did ?? '',
    country: m.country ?? '',
    effective_from: dateToAmino(m.effectiveFrom),
    effective_until: dateToAmino(m.effectiveUntil),
    validation_fees: u64ToStr(m.validationFees),
    issuance_fees: u64ToStr(m.issuanceFees),
    verification_fees: u64ToStr(m.verificationFees),
  }),
  fromAmino: (a: any): MsgCreateRootPermission =>
    MsgCreateRootPermission.fromPartial({
      creator: a.creator ?? '',
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      did: a.did ?? '',
      country: a.country ?? '',
      effectiveFrom: dateFromAmino(a.effective_from),
      effectiveUntil: dateFromAmino(a.effective_until),
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      issuanceFees: strToU64(a.issuance_fees) != null ? Number(strToU64(a.issuance_fees)!.toString()) : 0,
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
    }),
};

export const MsgCreatePermissionAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgCreatePermission',
  toAmino: (m: MsgCreatePermission) => clean({
    creator: m.creator ?? '',
    schema_id: u64ToStr(m.schemaId),
    type: m.type ?? PermissionType.UNSPECIFIED,
    did: m.did ?? '',
    country: m.country ?? '',
    effective_from: dateToAmino(m.effectiveFrom),
    effective_until: dateToAmino(m.effectiveUntil),
    verification_fees: u64ToStrIfNonZero(m.verificationFees),
    validation_fees: u64ToStrIfNonZero(m.validationFees),
  }),
  fromAmino: (a: any): MsgCreatePermission =>
    MsgCreatePermission.fromPartial({
      creator: a.creator ?? '',
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      type: a.type ?? PermissionType.UNSPECIFIED,
      did: a.did ?? '',
      country: a.country ?? '',
      effectiveFrom: dateFromAmino(a.effective_from),
      effectiveUntil: dateFromAmino(a.effective_until),
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
    }),
};

export const MsgExtendPermissionAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgExtendPermission',
  toAmino: (m: MsgExtendPermission) => clean({
    creator: m.creator ?? '',
    id: u64ToStr(m.id),
    effective_until: dateToAmino(m.effectiveUntil),
  }),
  fromAmino: (a: any): MsgExtendPermission =>
    MsgExtendPermission.fromPartial({
      creator: a.creator ?? '',
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      effectiveUntil: dateFromAmino(a.effective_until),
    }),
};

export const MsgRevokePermissionAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgRevokePermission',
  toAmino: (m: MsgRevokePermission) => clean({
    creator: m.creator ?? '',
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRevokePermission =>
    MsgRevokePermission.fromPartial({
      creator: a.creator ?? '',
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgStartPermissionVPAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgStartPermissionVP',
  toAmino: (m: MsgStartPermissionVP) => clean({
    creator: m.creator ?? '',
    type: m.type ?? PermissionType.UNSPECIFIED,
    validator_perm_id: u64ToStr(m.validatorPermId),
    country: m.country ?? '',
    did: m.did ?? '',
  }),
  fromAmino: (a: any): MsgStartPermissionVP =>
    MsgStartPermissionVP.fromPartial({
      creator: a.creator ?? '',
      type: a.type ?? PermissionType.UNSPECIFIED,
      validatorPermId: strToU64(a.validator_perm_id) != null ? Number(strToU64(a.validator_perm_id)!.toString()) : 0,
      country: a.country ?? '',
      did: a.did ?? '',
    }),
};

export const MsgRenewPermissionVPAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgRenewPermissionVP',
  toAmino: (m: MsgRenewPermissionVP) => clean({
    creator: m.creator ?? '',
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRenewPermissionVP =>
    MsgRenewPermissionVP.fromPartial({
      creator: a.creator ?? '',
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgSetPermissionVPToValidatedAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgSetPermissionVPToValidated',
  toAmino: (m: MsgSetPermissionVPToValidated) => clean({
    creator: m.creator ?? '',
    id: u64ToStr(m.id),
    effective_until: dateToAmino(m.effectiveUntil),
    validation_fees: u64ToStr(m.validationFees),
    issuance_fees: u64ToStr(m.issuanceFees),
    verification_fees: u64ToStr(m.verificationFees),
    country: m.country ?? '',
    vp_summary_digest_sri: m.vpSummaryDigestSri ?? '',
  }),
  fromAmino: (a: any): MsgSetPermissionVPToValidated =>
    MsgSetPermissionVPToValidated.fromPartial({
      creator: a.creator ?? '',
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      effectiveUntil: dateFromAmino(a.effective_until),
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      issuanceFees: strToU64(a.issuance_fees) != null ? Number(strToU64(a.issuance_fees)!.toString()) : 0,
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      country: a.country ?? '',
      vpSummaryDigestSri: a.vp_summary_digest_sri ?? '',
    }),
};

export const MsgCancelPermissionVPLastRequestAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgCancelPermissionVPLastRequest',
  toAmino: (m: MsgCancelPermissionVPLastRequest) => clean({
    creator: m.creator ?? '',
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgCancelPermissionVPLastRequest =>
    MsgCancelPermissionVPLastRequest.fromPartial({
      creator: a.creator ?? '',
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgCreateOrUpdatePermissionSessionAminoConverter: AminoConverter = {
  aminoType: '/verana.perm.v1.MsgCreateOrUpdatePermissionSession',
  toAmino: (m: MsgCreateOrUpdatePermissionSession) => clean({
    creator: m.creator ?? '',
    id: m.id ?? '',
    issuer_perm_id: u64ToStr(m.issuerPermId),
    verifier_perm_id: u64ToStr(m.verifierPermId),
    agent_perm_id: u64ToStr(m.agentPermId),
    wallet_agent_perm_id: u64ToStr(m.walletAgentPermId),
  }),
  fromAmino: (a: any): MsgCreateOrUpdatePermissionSession =>
    MsgCreateOrUpdatePermissionSession.fromPartial({
      creator: a.creator ?? '',
      id: a.id ?? '',
      issuerPermId: strToU64(a.issuer_perm_id) != null ? Number(strToU64(a.issuer_perm_id)!.toString()) : 0,
      verifierPermId: strToU64(a.verifier_perm_id) != null ? Number(strToU64(a.verifier_perm_id)!.toString()) : 0,
      agentPermId: strToU64(a.agent_perm_id) != null ? Number(strToU64(a.agent_perm_id)!.toString()) : 0,
      walletAgentPermId: strToU64(a.wallet_agent_perm_id) != null ? Number(strToU64(a.wallet_agent_perm_id)!.toString()) : 0,
    }),
};
