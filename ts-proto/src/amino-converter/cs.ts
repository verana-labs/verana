import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgArchiveCredentialSchema,
  MsgCreateCredentialSchema,
  MsgCreateSchemaAuthorizationPolicy,
  MsgIncreaseActiveSchemaAuthorizationPolicyVersion,
  MsgRevokeSchemaAuthorizationPolicy,
  MsgUpdateCredentialSchema,
} from "../codec/verana/cs/v1/tx";
import {
  clean,
  dateToIsoAmino,
  fromOptU32Amino,
  isoToDate,
  strToU64,
  toOptU32Amino,
  u32ToAmino,
  u64ToStr,
} from "./util/helpers";

export const MsgCreateCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: "verana/x/cs/MsgCreateCredentialSchema",
  toAmino: (m: MsgCreateCredentialSchema) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    ecosystem_id: u64ToStr(m.ecosystemId),
    json_schema: m.jsonSchema ?? "",
    issuer_grantor_validation_validity_period: toOptU32Amino(m.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(m.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(m.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(m.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(m.holderValidationValidityPeriod),
    issuer_onboarding_mode: u32ToAmino(m.issuerOnboardingMode),
    verifier_onboarding_mode: u32ToAmino(m.verifierOnboardingMode),
    holder_onboarding_mode: u32ToAmino(m.holderOnboardingMode),
    pricing_asset_type: m.pricingAssetType ?? 0,
    pricing_asset: m.pricingAsset ?? "",
    digest_algorithm: m.digestAlgorithm ?? "",
  }),
  fromAmino: (a: any): MsgCreateCredentialSchema =>
    MsgCreateCredentialSchema.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      ecosystemId: strToU64(a.ecosystem_id) != null ? Number(strToU64(a.ecosystem_id)!.toString()) : 0,
      jsonSchema: a.json_schema ?? "",
      issuerGrantorValidationValidityPeriod: fromOptU32Amino(a.issuer_grantor_validation_validity_period),
      verifierGrantorValidationValidityPeriod: fromOptU32Amino(a.verifier_grantor_validation_validity_period),
      issuerValidationValidityPeriod: fromOptU32Amino(a.issuer_validation_validity_period),
      verifierValidationValidityPeriod: fromOptU32Amino(a.verifier_validation_validity_period),
      holderValidationValidityPeriod: fromOptU32Amino(a.holder_validation_validity_period),
      issuerOnboardingMode: a.issuer_onboarding_mode ?? 0,
      verifierOnboardingMode: a.verifier_onboarding_mode ?? 0,
      holderOnboardingMode: a.holder_onboarding_mode ?? 0,
      pricingAssetType: a.pricing_asset_type ?? 0,
      pricingAsset: a.pricing_asset ?? "",
      digestAlgorithm: a.digest_algorithm ?? "",
    }),
};

export const MsgUpdateCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: "verana/x/cs/MsgUpdateCredentialSchema",
  toAmino: (m: MsgUpdateCredentialSchema) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    issuer_grantor_validation_validity_period: toOptU32Amino(m.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(m.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(m.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(m.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(m.holderValidationValidityPeriod),
  }),
  fromAmino: (a: any) =>
    MsgUpdateCredentialSchema.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      issuerGrantorValidationValidityPeriod: fromOptU32Amino(a.issuer_grantor_validation_validity_period),
      verifierGrantorValidationValidityPeriod: fromOptU32Amino(a.verifier_grantor_validation_validity_period),
      issuerValidationValidityPeriod: fromOptU32Amino(a.issuer_validation_validity_period),
      verifierValidationValidityPeriod: fromOptU32Amino(a.verifier_validation_validity_period),
      holderValidationValidityPeriod: fromOptU32Amino(a.holder_validation_validity_period),
    }),
};

export const MsgArchiveCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: "verana/x/cs/MsgArchiveCredentialSchema",
  toAmino: (m: MsgArchiveCredentialSchema) => ({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    archive: m.archive ?? false,
  }),
  fromAmino: (a: any): MsgArchiveCredentialSchema =>
    MsgArchiveCredentialSchema.fromPartial({
      corporation: a.corporation,
      operator: a.operator,
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      archive: a.archive ?? false,
    }),
};

export const MsgCreateSchemaAuthorizationPolicyAminoConverter: AminoConverter = {
  aminoType: "verana/x/cs/MsgCreateSchemaAuthPolicy",
  // [MOD-CS-MSG-5-1] Spec v4 draft 13: parameters are corporation, operator,
  // schema_id, role, url, digest_sri. effective_from/until are set to null at creation.
  toAmino: (m: MsgCreateSchemaAuthorizationPolicy) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    schema_id: u64ToStr(m.schemaId),
    role: m.role ?? 0,
    url: m.url ?? "",
    digest_sri: m.digestSri ?? "",
  }),
  fromAmino: (a: any): MsgCreateSchemaAuthorizationPolicy =>
    MsgCreateSchemaAuthorizationPolicy.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      role: a.role ?? 0,
      url: a.url ?? "",
      digestSri: a.digest_sri ?? "",
    }),
};

export const MsgIncreaseActiveSchemaAuthorizationPolicyVersionAminoConverter: AminoConverter = {
  aminoType: "verana/x/cs/MsgIncSchemaAuthPolicyVer",
  toAmino: (m: MsgIncreaseActiveSchemaAuthorizationPolicyVersion) => ({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    schema_id: u64ToStr(m.schemaId),
    role: m.role ?? 0,
  }),
  fromAmino: (a: any): MsgIncreaseActiveSchemaAuthorizationPolicyVersion =>
    MsgIncreaseActiveSchemaAuthorizationPolicyVersion.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      role: a.role ?? 0,
    }),
};

export const MsgRevokeSchemaAuthorizationPolicyAminoConverter: AminoConverter = {
  aminoType: "verana/x/cs/MsgRevokeSchemaAuthPolicy",
  toAmino: (m: MsgRevokeSchemaAuthorizationPolicy) => ({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    schema_id: u64ToStr(m.schemaId),
    role: m.role ?? 0,
    version: m.version ?? 0,
  }),
  fromAmino: (a: any): MsgRevokeSchemaAuthorizationPolicy =>
    MsgRevokeSchemaAuthorizationPolicy.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      role: a.role ?? 0,
      version: a.version ?? 0,
    }),
};
