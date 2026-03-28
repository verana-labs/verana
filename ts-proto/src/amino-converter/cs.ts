import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgArchiveCredentialSchema,
  MsgCreateCredentialSchema,
  MsgUpdateCredentialSchema,
} from "../codec/verana/cs/v1/tx";
import {
  clean,
  fromOptU32Amino,
  strToU64,
  toOptU32Amino,
  u32ToAmino,
  u64ToStr,
} from "./util/helpers";

export const MsgCreateCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: "/verana.cs.v1.MsgCreateCredentialSchema",
  toAmino: (m: MsgCreateCredentialSchema) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    tr_id: u64ToStr(m.trId),
    json_schema: m.jsonSchema ?? "",
    issuer_grantor_validation_validity_period: toOptU32Amino(m.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(m.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(m.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(m.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(m.holderValidationValidityPeriod),
    issuer_perm_management_mode: u32ToAmino(m.issuerPermManagementMode),
    verifier_perm_management_mode: u32ToAmino(m.verifierPermManagementMode),
    pricing_asset_type: m.pricingAssetType ?? 0,
    pricing_asset: m.pricingAsset ?? "",
    digest_algorithm: m.digestAlgorithm ?? "",
  }),
  fromAmino: (a: any): MsgCreateCredentialSchema =>
    MsgCreateCredentialSchema.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      trId: strToU64(a.tr_id) != null ? Number(strToU64(a.tr_id)!.toString()) : 0,
      jsonSchema: a.json_schema ?? "",
      issuerGrantorValidationValidityPeriod: fromOptU32Amino(a.issuer_grantor_validation_validity_period),
      verifierGrantorValidationValidityPeriod: fromOptU32Amino(a.verifier_grantor_validation_validity_period),
      issuerValidationValidityPeriod: fromOptU32Amino(a.issuer_validation_validity_period),
      verifierValidationValidityPeriod: fromOptU32Amino(a.verifier_validation_validity_period),
      holderValidationValidityPeriod: fromOptU32Amino(a.holder_validation_validity_period),
      issuerPermManagementMode: a.issuer_perm_management_mode ?? 0,
      verifierPermManagementMode: a.verifier_perm_management_mode ?? 0,
      pricingAssetType: a.pricing_asset_type ?? 0,
      pricingAsset: a.pricing_asset ?? "",
      digestAlgorithm: a.digest_algorithm ?? "",
    }),
};

export const MsgUpdateCredentialSchemaAminoConverter: AminoConverter = {
  aminoType: "/verana.cs.v1.MsgUpdateCredentialSchema",
  toAmino: (m: MsgUpdateCredentialSchema) => clean({
    authority: m.authority ?? "",
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
      authority: a.authority ?? "",
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
  aminoType: "/verana.cs.v1.MsgArchiveCredentialSchema",
  toAmino: (m: MsgArchiveCredentialSchema) => ({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    archive: m.archive ?? false,
  }),
  fromAmino: (a: any): MsgArchiveCredentialSchema =>
    MsgArchiveCredentialSchema.fromPartial({
      authority: a.authority,
      operator: a.operator,
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      archive: a.archive ?? false,
    }),
};
