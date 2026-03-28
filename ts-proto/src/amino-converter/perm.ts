import type { AminoConverter } from "@cosmjs/stargate";
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
} from "../codec/verana/perm/v1/tx";
import { PermissionType } from "../codec/verana/perm/v1/types";
import {
  aminoToDuration,
  clean,
  dateToIsoAmino,
  durationToAmino,
  isoToDate,
  strToU64,
  u64ToStr,
  u64ToStrIfNonZero,
} from "./util/helpers";

export const MsgCreateRootPermissionAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgCreateRootPermission",
  toAmino: (m: MsgCreateRootPermission) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    schema_id: u64ToStr(m.schemaId),
    did: m.did ?? "",
    effective_from: dateToIsoAmino(m.effectiveFrom),
    effective_until: dateToIsoAmino(m.effectiveUntil),
    validation_fees: u64ToStr(m.validationFees),
    issuance_fees: u64ToStr(m.issuanceFees),
    verification_fees: u64ToStr(m.verificationFees),
  }),
  fromAmino: (a: any): MsgCreateRootPermission =>
    MsgCreateRootPermission.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      did: a.did ?? "",
      effectiveFrom: isoToDate(a.effective_from),
      effectiveUntil: isoToDate(a.effective_until),
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      issuanceFees: strToU64(a.issuance_fees) != null ? Number(strToU64(a.issuance_fees)!.toString()) : 0,
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
    }),
};

export const MsgAdjustPermissionAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgAdjustPermission",
  toAmino: (m: MsgAdjustPermission) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    effective_until: dateToIsoAmino(m.effectiveUntil),
  }),
  fromAmino: (a: any): MsgAdjustPermission =>
    MsgAdjustPermission.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      effectiveUntil: isoToDate(a.effective_until),
    }),
};

export const MsgRevokePermissionAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgRevokePermission",
  toAmino: (m: MsgRevokePermission) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRevokePermission =>
    MsgRevokePermission.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgStartPermissionVPAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgStartPermissionVP",
  toAmino: (m: MsgStartPermissionVP) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    type: m.type ?? PermissionType.UNSPECIFIED,
    validator_perm_id: u64ToStr(m.validatorPermId),
    did: m.did ?? "",
    validation_fees: m.validationFees ? { value: u64ToStr(m.validationFees.value) } : undefined,
    issuance_fees: m.issuanceFees ? { value: u64ToStr(m.issuanceFees.value) } : undefined,
    verification_fees: m.verificationFees ? { value: u64ToStr(m.verificationFees.value) } : undefined,
    vs_operator: m.vsOperator || undefined,
    vs_operator_authz_enabled: m.vsOperatorAuthzEnabled || undefined,
    vs_operator_authz_spend_limit: m.vsOperatorAuthzSpendLimit ?? [],
    vs_operator_authz_with_feegrant: m.vsOperatorAuthzWithFeegrant || undefined,
    vs_operator_authz_fee_spend_limit: m.vsOperatorAuthzFeeSpendLimit ?? [],
    vs_operator_authz_spend_period: durationToAmino(m.vsOperatorAuthzSpendPeriod),
  }),
  fromAmino: (a: any): MsgStartPermissionVP =>
    MsgStartPermissionVP.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      type: a.type ?? PermissionType.UNSPECIFIED,
      validatorPermId: strToU64(a.validator_perm_id) != null ? Number(strToU64(a.validator_perm_id)!.toString()) : 0,
      did: a.did ?? "",
      validationFees: a.validation_fees ? { value: Number(a.validation_fees.value ?? a.validation_fees) } : undefined,
      issuanceFees: a.issuance_fees ? { value: Number(a.issuance_fees.value ?? a.issuance_fees) } : undefined,
      verificationFees: a.verification_fees ? { value: Number(a.verification_fees.value ?? a.verification_fees) } : undefined,
      vsOperator: a.vs_operator ?? "",
      vsOperatorAuthzEnabled: a.vs_operator_authz_enabled ?? false,
      vsOperatorAuthzSpendLimit: a.vs_operator_authz_spend_limit ?? [],
      vsOperatorAuthzWithFeegrant: a.vs_operator_authz_with_feegrant ?? false,
      vsOperatorAuthzFeeSpendLimit: a.vs_operator_authz_fee_spend_limit ?? [],
      vsOperatorAuthzSpendPeriod: aminoToDuration(a.vs_operator_authz_spend_period),
    }),
};

export const MsgRenewPermissionVPAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgRenewPermissionVP",
  toAmino: (m: MsgRenewPermissionVP) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRenewPermissionVP =>
    MsgRenewPermissionVP.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgSetPermissionVPToValidatedAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgSetPermissionVPToValidated",
  toAmino: (m: MsgSetPermissionVPToValidated) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    effective_until: dateToIsoAmino(m.effectiveUntil),
    validation_fees: u64ToStr(m.validationFees),
    issuance_fees: u64ToStr(m.issuanceFees),
    verification_fees: u64ToStr(m.verificationFees),
    vp_summary_digest_sri: m.vpSummaryDigestSri ?? "",
    issuance_fee_discount: u64ToStr(m.issuanceFeeDiscount),
    verification_fee_discount: u64ToStr(m.verificationFeeDiscount),
  }),
  fromAmino: (a: any): MsgSetPermissionVPToValidated =>
    MsgSetPermissionVPToValidated.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      effectiveUntil: isoToDate(a.effective_until),
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      issuanceFees: strToU64(a.issuance_fees) != null ? Number(strToU64(a.issuance_fees)!.toString()) : 0,
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      vpSummaryDigestSri: a.vp_summary_digest_sri ?? "",
      issuanceFeeDiscount: strToU64(a.issuance_fee_discount) != null
        ? Number(strToU64(a.issuance_fee_discount)!.toString())
        : 0,
      verificationFeeDiscount: strToU64(a.verification_fee_discount) != null
        ? Number(strToU64(a.verification_fee_discount)!.toString())
        : 0,
    }),
};

export const MsgCancelPermissionVPLastRequestAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgCancelPermissionVPLastRequest",
  toAmino: (m: MsgCancelPermissionVPLastRequest) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgCancelPermissionVPLastRequest =>
    MsgCancelPermissionVPLastRequest.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgCreateOrUpdatePermissionSessionAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgCreateOrUpdatePermissionSession",
  toAmino: (m: MsgCreateOrUpdatePermissionSession) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: m.id ?? "",
    issuer_perm_id: u64ToStr(m.issuerPermId),
    verifier_perm_id: u64ToStr(m.verifierPermId),
    agent_perm_id: u64ToStr(m.agentPermId),
    wallet_agent_perm_id: u64ToStr(m.walletAgentPermId),
    digest: m.digest ?? undefined,
  }),
  fromAmino: (a: any): MsgCreateOrUpdatePermissionSession =>
    MsgCreateOrUpdatePermissionSession.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: a.id ?? "",
      issuerPermId: strToU64(a.issuer_perm_id) != null ? Number(strToU64(a.issuer_perm_id)!.toString()) : 0,
      verifierPermId: strToU64(a.verifier_perm_id) != null ? Number(strToU64(a.verifier_perm_id)!.toString()) : 0,
      agentPermId: strToU64(a.agent_perm_id) != null ? Number(strToU64(a.agent_perm_id)!.toString()) : 0,
      walletAgentPermId: strToU64(a.wallet_agent_perm_id) != null
        ? Number(strToU64(a.wallet_agent_perm_id)!.toString())
        : 0,
      digest: a.digest ?? "",
    }),
};

export const MsgSlashPermissionTrustDepositAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgSlashPermissionTrustDeposit",
  toAmino: (m: MsgSlashPermissionTrustDeposit) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    amount: u64ToStr(m.amount),
  }),
  fromAmino: (a: any): MsgSlashPermissionTrustDeposit =>
    MsgSlashPermissionTrustDeposit.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      amount: strToU64(a.amount) != null ? Number(strToU64(a.amount)!.toString()) : 0,
    }),
};

export const MsgRepayPermissionSlashedTrustDepositAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgRepayPermissionSlashedTrustDeposit",
  toAmino: (m: MsgRepayPermissionSlashedTrustDeposit) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRepayPermissionSlashedTrustDeposit =>
    MsgRepayPermissionSlashedTrustDeposit.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgCreatePermissionAminoConverter: AminoConverter = {
  aminoType: "/verana.perm.v1.MsgCreatePermission",
  toAmino: (m: MsgCreatePermission) => clean({
    authority: m.authority ?? "",
    operator: m.operator ?? "",
    type: m.type ?? 0,
    validator_perm_id: u64ToStr(m.validatorPermId),
    did: m.did ?? "",
    effective_from: dateToIsoAmino(m.effectiveFrom),
    effective_until: dateToIsoAmino(m.effectiveUntil),
    verification_fees: u64ToStrIfNonZero(m.verificationFees),
    validation_fees: u64ToStrIfNonZero(m.validationFees),
    vs_operator: m.vsOperator || undefined,
    vs_operator_authz_enabled: m.vsOperatorAuthzEnabled || undefined,
    vs_operator_authz_spend_limit: m.vsOperatorAuthzSpendLimit ?? [],
    vs_operator_authz_with_feegrant: m.vsOperatorAuthzWithFeegrant || undefined,
    vs_operator_authz_fee_spend_limit: m.vsOperatorAuthzFeeSpendLimit ?? [],
    vs_operator_authz_spend_period: durationToAmino(m.vsOperatorAuthzSpendPeriod),
  }),
  fromAmino: (a: any): MsgCreatePermission =>
    MsgCreatePermission.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      type: a.type ?? 0,
      validatorPermId: strToU64(a.validator_perm_id) != null ? Number(strToU64(a.validator_perm_id)!.toString()) : 0,
      did: a.did ?? "",
      effectiveFrom: isoToDate(a.effective_from),
      effectiveUntil: isoToDate(a.effective_until),
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      vsOperator: a.vs_operator ?? "",
      vsOperatorAuthzEnabled: a.vs_operator_authz_enabled ?? false,
      vsOperatorAuthzSpendLimit: a.vs_operator_authz_spend_limit ?? [],
      vsOperatorAuthzWithFeegrant: a.vs_operator_authz_with_feegrant ?? false,
      vsOperatorAuthzFeeSpendLimit: a.vs_operator_authz_fee_spend_limit ?? [],
      vsOperatorAuthzSpendPeriod: aminoToDuration(a.vs_operator_authz_spend_period),
    }),
};
