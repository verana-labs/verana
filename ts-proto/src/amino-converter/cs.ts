import {
  MsgCreateCredentialSchema,
  MsgUpdateCredentialSchema,
  MsgArchiveCredentialSchema,
} from '../codec/verana/cs/v1/tx';
import { u64ToStr, strToU64, u32ToAmino, fromOptU32Amino, toOptU32Amino, clean } from './util/helpers';

/**
 * Amino converter for MsgCreateCredentialSchema
 */
export const MsgCreateCredentialSchemaAminoConverter = {
  aminoType: '/verana.cs.v1.MsgCreateCredentialSchema',
  // Proto → Amino JSON
  toAmino: (msg: MsgCreateCredentialSchema) => clean({
    creator: msg.creator ?? '',
    tr_id: u64ToStr(msg.trId), // uint64 -> string
    json_schema: msg.jsonSchema ?? '',
    issuer_grantor_validation_validity_period: toOptU32Amino(msg.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(msg.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(msg.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(msg.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(msg.holderValidationValidityPeriod),
    issuer_perm_management_mode: u32ToAmino(msg.issuerPermManagementMode) , // uint32 -> number
    verifier_perm_management_mode: u32ToAmino(msg.verifierPermManagementMode) , // uint32 -> number
  }),
  // Amino JSON → Proto
  fromAmino: (value: any ): MsgCreateCredentialSchema =>
    MsgCreateCredentialSchema.fromPartial({
      creator: value.creator ?? '',
      trId: strToU64(value.tr_id),
      jsonSchema: value.json_schema ?? '',
      issuerGrantorValidationValidityPeriod: fromOptU32Amino(value.issuer_grantor_validation_validity_period),
      verifierGrantorValidationValidityPeriod: fromOptU32Amino(value.verifier_grantor_validation_validity_period),
      issuerValidationValidityPeriod: fromOptU32Amino(value.issuer_validation_validity_period),
      verifierValidationValidityPeriod: fromOptU32Amino(value.verifier_validation_validity_period),
      holderValidationValidityPeriod: fromOptU32Amino(value.holder_validation_validity_period),
      issuerPermManagementMode: (value.issuer_perm_management_mode) ,
      verifierPermManagementMode: (value.verifier_perm_management_mode) ,
    }),
};

/**
 * Amino converter for MsgUpdateCredentialSchema
 */
export const MsgUpdateCredentialSchemaAminoConverter = {
  aminoType: '/verana.cs.v1.MsgUpdateCredentialSchema',
  toAmino: (msg: MsgUpdateCredentialSchema) => clean({
    creator: msg.creator ?? '',
    id: u64ToStr(msg.id),
    issuer_grantor_validation_validity_period: toOptU32Amino(msg.issuerGrantorValidationValidityPeriod),
    verifier_grantor_validation_validity_period: toOptU32Amino(msg.verifierGrantorValidationValidityPeriod),
    issuer_validation_validity_period: toOptU32Amino(msg.issuerValidationValidityPeriod),
    verifier_validation_validity_period: toOptU32Amino(msg.verifierValidationValidityPeriod),
    holder_validation_validity_period: toOptU32Amino(msg.holderValidationValidityPeriod),
  }),
  fromAmino: (value: any) => MsgUpdateCredentialSchema.fromPartial({
    creator: value.creator ?? '',
    id: strToU64(value.id),
    issuerGrantorValidationValidityPeriod: fromOptU32Amino(value.issuer_grantor_validation_validity_period),
    verifierGrantorValidationValidityPeriod: fromOptU32Amino(value.verifier_grantor_validation_validity_period),
    issuerValidationValidityPeriod: fromOptU32Amino(value.issuer_validation_validity_period),
    verifierValidationValidityPeriod: fromOptU32Amino(value.verifier_validation_validity_period),
    holderValidationValidityPeriod: fromOptU32Amino(value.holder_validation_validity_period),
  }),
};

/**
 * Amino converter for MsgArchiveCredentialSchema
 */
export const MsgArchiveCredentialSchemaAminoConverter = {
  aminoType: '/verana.cs.v1.MsgArchiveCredentialSchema',
  // Proto → Amino JSON
  toAmino: (msg: MsgArchiveCredentialSchema) => clean({
    creator: msg.creator ?? '',
    id: u64ToStr(msg.id),
    archive: msg.archive ? true : undefined, // omit if false
  }),
  // Amino JSON → Proto
  fromAmino: (value: any): MsgArchiveCredentialSchema =>
    MsgArchiveCredentialSchema.fromPartial({
      creator: value.creator,
      id: strToU64(value.id),
      archive: value.archive ?? false,
    }),
};
