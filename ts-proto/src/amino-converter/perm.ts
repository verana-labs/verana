'use client';

import {
  MsgStartPermissionVP,
  MsgRenewPermissionVP,
  MsgSetPermissionVPToValidated,
  MsgCancelPermissionVPLastRequest,
  MsgCreateRootPermission,
  MsgExtendPermission,
  MsgRevokePermission,
  MsgCreateOrUpdatePermissionSession,
  MsgSlashPermissionTrustDeposit,
  MsgRepayPermissionSlashedTrustDeposit,
  MsgCreatePermission,
} from '../codec/verana/perm/v1/tx';

import { clean, strToU64, u64ToStr, u64ToStrIfNonZero, dateToIsoAmino, isoToDate } from './util/helpers';

/**
 * Amino converter for MsgStartPermissionVP
 */
export const MsgStartPermissionVPAminoConverter = {
  aminoType: '/verana.perm.v1.MsgStartPermissionVP',
  toAmino: (msg: MsgStartPermissionVP) => ({
    creator: msg.creator,
    type: msg.type,
    validator_perm_id: u64ToStr(msg.validatorPermId), // uint64 -> string
    country: msg.country,
    did: msg.did,
  }),
  fromAmino: (value: any) =>
    MsgStartPermissionVP.fromPartial({
      creator: value.creator,
      type: value.type,
      validatorPermId: strToU64(value.validator_perm_id), // string -> Long (uint64)
      country: value.country,
      did: value.did,
    }),
};

/**
 * Amino converter for MsgRenewPermissionVP
 */
export const MsgRenewPermissionVPAminoConverter = {
  aminoType: '/verana.perm.v1.MsgRenewPermissionVP',
  toAmino: (msg: MsgRenewPermissionVP) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgRenewPermissionVP.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgSetPermissionVPToValidated
 */
export const MsgSetPermissionVPToValidatedAminoConverter = {
  aminoType: '/verana.perm.v1.MsgSetPermissionVPToValidated',
  toAmino: (msg: MsgSetPermissionVPToValidated) => clean({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
    effective_until: dateToIsoAmino(msg.effectiveUntil), // Date -> ISO string (optional)
    validation_fees: u64ToStr(msg.validationFees), // uint64 -> string
    issuance_fees: u64ToStr(msg.issuanceFees), // uint64 -> string
    verification_fees: u64ToStr(msg.verificationFees), // uint64 -> string
    country: msg.country,
    vp_summary_digest_sri: msg.vpSummaryDigestSri,
    issuance_fee_discount: u64ToStrIfNonZero(msg.issuanceFeeDiscount), // uint64 -> string, omit if zero
    verification_fee_discount: u64ToStrIfNonZero(msg.verificationFeeDiscount), // uint64 -> string, omit if zero
  }),
  fromAmino: (value: any) =>
    MsgSetPermissionVPToValidated.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
      effectiveUntil: isoToDate(value.effective_until), // ISO string -> Date (optional)
      validationFees: strToU64(value.validation_fees), // string -> Long (uint64)
      issuanceFees: strToU64(value.issuance_fees), // string -> Long (uint64)
      verificationFees: strToU64(value.verification_fees), // string -> Long (uint64)
      country: value.country,
      vpSummaryDigestSri: value.vp_summary_digest_sri,
      issuanceFeeDiscount: strToU64(value.issuance_fee_discount), // string -> Long (uint64)
      verificationFeeDiscount: strToU64(value.verification_fee_discount), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgCancelPermissionVPLastRequest
 */
export const MsgCancelPermissionVPLastRequestAminoConverter = {
  aminoType: '/verana.perm.v1.MsgCancelPermissionVPLastRequest',
  toAmino: (msg: MsgCancelPermissionVPLastRequest) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgCancelPermissionVPLastRequest.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgCreateRootPermission
 */
export const MsgCreateRootPermissionAminoConverter = {
  aminoType: '/verana.perm.v1.MsgCreateRootPermission',
  toAmino: (msg: MsgCreateRootPermission) => ({
    creator: msg.creator,
    schema_id: u64ToStr(msg.schemaId), // uint64 -> string
    did: msg.did,
    country: msg.country,
    effective_from: dateToIsoAmino(msg.effectiveFrom), // Date -> ISO string (optional)
    effective_until: dateToIsoAmino(msg.effectiveUntil), // Date -> ISO string (optional)
    validation_fees: u64ToStr(msg.validationFees), // uint64 -> string
    issuance_fees: u64ToStr(msg.issuanceFees), // uint64 -> string
    verification_fees: u64ToStr(msg.verificationFees), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgCreateRootPermission.fromPartial({
      creator: value.creator,
      schemaId: strToU64(value.schema_id), // string -> Long (uint64)
      did: value.did,
      country: value.country,
      effectiveFrom: isoToDate(value.effective_from), // ISO string -> Date (optional)
      effectiveUntil: isoToDate(value.effective_until), // ISO string -> Date (optional)
      validationFees: strToU64(value.validation_fees), // string -> Long (uint64)
      issuanceFees: strToU64(value.issuance_fees), // string -> Long (uint64)
      verificationFees: strToU64(value.verification_fees), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgExtendPermission
 */
export const MsgExtendPermissionAminoConverter = {
  aminoType: '/verana.perm.v1.MsgExtendPermission',
  toAmino: (msg: MsgExtendPermission) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
    effective_until: dateToIsoAmino(msg.effectiveUntil), // Date -> ISO string (optional)
  }),
  fromAmino: (value: any) =>
    MsgExtendPermission.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
      effectiveUntil: isoToDate(value.effective_until), // ISO string -> Date (optional)
    }),
};

/**
 * Amino converter for MsgRevokePermission
 */
export const MsgRevokePermissionAminoConverter = {
  aminoType: '/verana.perm.v1.MsgRevokePermission',
  toAmino: (msg: MsgRevokePermission) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgRevokePermission.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgCreateOrUpdatePermissionSession
 */
export const MsgCreateOrUpdatePermissionSessionAminoConverter = {
  aminoType: '/verana.perm.v1.MsgCreateOrUpdatePermissionSession',
  toAmino: (msg: MsgCreateOrUpdatePermissionSession) => ({
    creator: msg.creator,
    id: msg.id, // UUID string
    issuer_perm_id: u64ToStr(msg.issuerPermId), // uint64 -> string
    verifier_perm_id: u64ToStr(msg.verifierPermId), // uint64 -> string
    agent_perm_id: u64ToStr(msg.agentPermId), // uint64 -> string
    wallet_agent_perm_id: u64ToStr(msg.walletAgentPermId), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgCreateOrUpdatePermissionSession.fromPartial({
      creator: value.creator,
      id: value.id,
      issuerPermId: strToU64(value.issuer_perm_id), // string -> Long (uint64)
      verifierPermId: strToU64(value.verifier_perm_id), // string -> Long (uint64)
      agentPermId: strToU64(value.agent_perm_id), // string -> Long (uint64)
      walletAgentPermId: strToU64(value.wallet_agent_perm_id), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgSlashPermissionTrustDeposit
 */
export const MsgSlashPermissionTrustDepositAminoConverter = {
  aminoType: '/verana.perm.v1.MsgSlashPermissionTrustDeposit',
  toAmino: (msg: MsgSlashPermissionTrustDeposit) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
    amount: u64ToStr(msg.amount), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgSlashPermissionTrustDeposit.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
      amount: strToU64(value.amount), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgRepayPermissionSlashedTrustDeposit
 */
export const MsgRepayPermissionSlashedTrustDepositAminoConverter = {
  aminoType: '/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit',
  toAmino: (msg: MsgRepayPermissionSlashedTrustDeposit) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgRepayPermissionSlashedTrustDeposit.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgCreatePermission
 */
export const MsgCreatePermissionAminoConverter = {
  aminoType: '/verana.perm.v1.MsgCreatePermission',
  toAmino: (msg: MsgCreatePermission) => clean({
    creator: msg.creator,
    schema_id: u64ToStr(msg.schemaId), // uint64 -> string
    type: msg.type,
    did: msg.did,
    country: msg.country,
    effective_from: dateToIsoAmino(msg.effectiveFrom), // Date -> ISO string (optional)
    effective_until: dateToIsoAmino(msg.effectiveUntil), // Date -> ISO string (optional)
    verification_fees: u64ToStrIfNonZero(msg.verificationFees), // uint64 -> string, omit if zero
    validation_fees: u64ToStrIfNonZero(msg.validationFees), // uint64 -> string, omit if zero
  }),
  fromAmino: (value: any) =>
    MsgCreatePermission.fromPartial({
      creator: value.creator,
      schemaId: strToU64(value.schema_id), // string -> Long (uint64)
      type: value.type,
      did: value.did,
      country: value.country,
      effectiveFrom: isoToDate(value.effective_from), // ISO string -> Date (optional)
      effectiveUntil: isoToDate(value.effective_until), // ISO string -> Date (optional)
      verificationFees: strToU64(value.verification_fees), // string -> Long (uint64)
      validationFees: strToU64(value.validation_fees), // string -> Long (uint64)
    }),
};
