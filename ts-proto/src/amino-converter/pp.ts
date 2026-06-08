import type { AminoConverter } from "@cosmjs/stargate";
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
} from "../codec/verana/pp/v1/tx";
import { ParticipantRole } from "../codec/verana/pp/v1/types";
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

export const MsgCreateRootParticipantAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgCreateRootParticipant",
  // [MOD-PP-MSG-7-3] spec v4 draft 13: perm.role is hardcoded to ECOSYSTEM.
  toAmino: (m: MsgCreateRootParticipant) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    schema_id: u64ToStr(m.schemaId),
    did: m.did ?? "",
    effective_from: dateToIsoAmino(m.effectiveFrom),
    effective_until: dateToIsoAmino(m.effectiveUntil),
    validation_fees: u64ToStr(m.validationFees),
    issuance_fees: u64ToStr(m.issuanceFees),
    verification_fees: u64ToStr(m.verificationFees),
    // VSOA params (proto fields 10-15). The chain's aminojson encodes empty
    // dont_omitempty Coin arrays as `null` (NOT []) and omits empty plain
    // repeated/scalar fields. The sign bytes must match exactly.
    vs_operator: m.vsOperator || undefined,
    vs_operator_authz_msg_types: m.vsOperatorAuthzMsgTypes?.length ? m.vsOperatorAuthzMsgTypes : undefined,
    vs_operator_authz_spend_limit: m.vsOperatorAuthzSpendLimit?.length ? m.vsOperatorAuthzSpendLimit : null,
    vs_operator_authz_with_feegrant: m.vsOperatorAuthzWithFeegrant || undefined,
    vs_operator_authz_fee_spend_limit: m.vsOperatorAuthzFeeSpendLimit?.length ? m.vsOperatorAuthzFeeSpendLimit : null,
    vs_operator_authz_period: durationToAmino(m.vsOperatorAuthzPeriod),
  }),
  fromAmino: (a: any): MsgCreateRootParticipant =>
    MsgCreateRootParticipant.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      schemaId: strToU64(a.schema_id) != null ? Number(strToU64(a.schema_id)!.toString()) : 0,
      did: a.did ?? "",
      effectiveFrom: isoToDate(a.effective_from),
      effectiveUntil: isoToDate(a.effective_until),
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      issuanceFees: strToU64(a.issuance_fees) != null ? Number(strToU64(a.issuance_fees)!.toString()) : 0,
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      vsOperator: a.vs_operator ?? "",
      vsOperatorAuthzMsgTypes: a.vs_operator_authz_msg_types ?? [],
      vsOperatorAuthzSpendLimit: a.vs_operator_authz_spend_limit ?? [],
      vsOperatorAuthzWithFeegrant: a.vs_operator_authz_with_feegrant ?? false,
      vsOperatorAuthzFeeSpendLimit: a.vs_operator_authz_fee_spend_limit ?? [],
      vsOperatorAuthzPeriod: aminoToDuration(a.vs_operator_authz_period),
    }),
};

export const MsgSetParticipantEffectiveUntilAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgSetPartEffectiveUntil",
  toAmino: (m: MsgSetParticipantEffectiveUntil) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    effective_until: dateToIsoAmino(m.effectiveUntil),
  }),
  fromAmino: (a: any): MsgSetParticipantEffectiveUntil =>
    MsgSetParticipantEffectiveUntil.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      effectiveUntil: isoToDate(a.effective_until),
    }),
};

export const MsgRevokeParticipantAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgRevokeParticipant",
  toAmino: (m: MsgRevokeParticipant) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRevokeParticipant =>
    MsgRevokeParticipant.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgStartParticipantOPAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgStartParticipantOP",
  toAmino: (m: MsgStartParticipantOP) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    role: m.role ?? ParticipantRole.UNSPECIFIED,
    validator_participant_id: u64ToStr(m.validatorParticipantId),
    did: m.did ?? "",
    validation_fees: m.validationFees ? { value: u64ToStr(m.validationFees.value) } : undefined,
    issuance_fees: m.issuanceFees ? { value: u64ToStr(m.issuanceFees.value) } : undefined,
    verification_fees: m.verificationFees ? { value: u64ToStr(m.verificationFees.value) } : undefined,
    vs_operator: m.vsOperator || undefined,
    vs_operator_authz_msg_types: m.vsOperatorAuthzMsgTypes?.length ? m.vsOperatorAuthzMsgTypes : undefined,
    vs_operator_authz_spend_limit: m.vsOperatorAuthzSpendLimit?.length ? m.vsOperatorAuthzSpendLimit : null,
    vs_operator_authz_with_feegrant: m.vsOperatorAuthzWithFeegrant || undefined,
    vs_operator_authz_fee_spend_limit: m.vsOperatorAuthzFeeSpendLimit?.length ? m.vsOperatorAuthzFeeSpendLimit : null,
    vs_operator_authz_period: durationToAmino(m.vsOperatorAuthzPeriod),
  }),
  fromAmino: (a: any): MsgStartParticipantOP =>
    MsgStartParticipantOP.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      role: a.role ?? ParticipantRole.UNSPECIFIED,
      validatorParticipantId: strToU64(a.validator_participant_id) != null ? Number(strToU64(a.validator_participant_id)!.toString()) : 0,
      did: a.did ?? "",
      validationFees: a.validation_fees ? { value: Number(a.validation_fees.value ?? a.validation_fees) } : undefined,
      issuanceFees: a.issuance_fees ? { value: Number(a.issuance_fees.value ?? a.issuance_fees) } : undefined,
      verificationFees: a.verification_fees ? { value: Number(a.verification_fees.value ?? a.verification_fees) } : undefined,
      vsOperator: a.vs_operator ?? "",
      vsOperatorAuthzMsgTypes: a.vs_operator_authz_msg_types ?? [],
      vsOperatorAuthzSpendLimit: a.vs_operator_authz_spend_limit ?? [],
      vsOperatorAuthzWithFeegrant: a.vs_operator_authz_with_feegrant ?? false,
      vsOperatorAuthzFeeSpendLimit: a.vs_operator_authz_fee_spend_limit ?? [],
      vsOperatorAuthzPeriod: aminoToDuration(a.vs_operator_authz_period),
    }),
};

export const MsgRenewParticipantOPAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgRenewParticipantOP",
  // [MOD-PP-MSG-2-1] spec v4 draft 13 parameters: corporation, operator, id.
  toAmino: (m: MsgRenewParticipantOP) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgRenewParticipantOP =>
    MsgRenewParticipantOP.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgSetParticipantOPToValidatedAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgSetPartOPValidated",
  toAmino: (m: MsgSetParticipantOPToValidated) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    effective_until: dateToIsoAmino(m.effectiveUntil),
    validation_fees: u64ToStr(m.validationFees),
    issuance_fees: u64ToStr(m.issuanceFees),
    verification_fees: u64ToStr(m.verificationFees),
    op_summary_digest: m.opSummaryDigest ?? "",
    issuance_fee_discount: u64ToStr(m.issuanceFeeDiscount),
    verification_fee_discount: u64ToStr(m.verificationFeeDiscount),
  }),
  fromAmino: (a: any): MsgSetParticipantOPToValidated =>
    MsgSetParticipantOPToValidated.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      effectiveUntil: isoToDate(a.effective_until),
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      issuanceFees: strToU64(a.issuance_fees) != null ? Number(strToU64(a.issuance_fees)!.toString()) : 0,
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      opSummaryDigest: a.op_summary_digest ?? "",
      issuanceFeeDiscount: strToU64(a.issuance_fee_discount) != null
        ? Number(strToU64(a.issuance_fee_discount)!.toString())
        : 0,
      verificationFeeDiscount: strToU64(a.verification_fee_discount) != null
        ? Number(strToU64(a.verification_fee_discount)!.toString())
        : 0,
    }),
};

export const MsgCancelParticipantOPLastRequestAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgCancelPartOPLastReq",
  toAmino: (m: MsgCancelParticipantOPLastRequest) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
  }),
  fromAmino: (a: any): MsgCancelParticipantOPLastRequest =>
    MsgCancelParticipantOPLastRequest.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
    }),
};

export const MsgCreateOrUpdateParticipantSessionAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgCreateOrUpdatePartSess",
  toAmino: (m: MsgCreateOrUpdateParticipantSession) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: m.id ?? "",
    issuer_participant_id: u64ToStr(m.issuerParticipantId),
    verifier_participant_id: u64ToStr(m.verifierParticipantId),
    agent_participant_id: u64ToStr(m.agentParticipantId),
    wallet_agent_participant_id: u64ToStr(m.walletAgentParticipantId),
    digest: m.digest ?? undefined,
  }),
  fromAmino: (a: any): MsgCreateOrUpdateParticipantSession =>
    MsgCreateOrUpdateParticipantSession.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: a.id ?? "",
      issuerParticipantId: strToU64(a.issuer_participant_id) != null ? Number(strToU64(a.issuer_participant_id)!.toString()) : 0,
      verifierParticipantId: strToU64(a.verifier_participant_id) != null ? Number(strToU64(a.verifier_participant_id)!.toString()) : 0,
      agentParticipantId: strToU64(a.agent_participant_id) != null ? Number(strToU64(a.agent_participant_id)!.toString()) : 0,
      walletAgentParticipantId: strToU64(a.wallet_agent_participant_id) != null
        ? Number(strToU64(a.wallet_agent_participant_id)!.toString())
        : 0,
      digest: a.digest ?? "",
    }),
};

export const MsgSlashParticipantTrustDepositAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgSlashParticipantTD",
  // [MOD-PP-MSG-12-1] spec v4 draft 13 adds mandatory reason.
  toAmino: (m: MsgSlashParticipantTrustDeposit) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    amount: u64ToStr(m.amount),
    reason: m.reason ?? "",
  }),
  fromAmino: (a: any): MsgSlashParticipantTrustDeposit =>
    MsgSlashParticipantTrustDeposit.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      amount: strToU64(a.amount) != null ? Number(strToU64(a.amount)!.toString()) : 0,
      reason: a.reason ?? "",
    }),
};

export const MsgRepayParticipantSlashedTrustDepositAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgRepayPartSlashedTD",
  toAmino: (m: MsgRepayParticipantSlashedTrustDeposit) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    id: u64ToStr(m.id),
    amount: u64ToStrIfNonZero(m.amount),
  }),
  fromAmino: (a: any): MsgRepayParticipantSlashedTrustDeposit =>
    MsgRepayParticipantSlashedTrustDeposit.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      amount: strToU64(a.amount) != null ? Number(strToU64(a.amount)!.toString()) : 0,
    }),
};

export const MsgSelfCreateParticipantAminoConverter: AminoConverter = {
  aminoType: "verana/x/pp/MsgSelfCreateParticipant",
  toAmino: (m: MsgSelfCreateParticipant) => clean({
    corporation: m.corporation ?? "",
    operator: m.operator ?? "",
    role: m.role ?? 0,
    validator_participant_id: u64ToStr(m.validatorParticipantId),
    did: m.did ?? "",
    effective_from: dateToIsoAmino(m.effectiveFrom),
    effective_until: dateToIsoAmino(m.effectiveUntil),
    verification_fees: u64ToStrIfNonZero(m.verificationFees),
    validation_fees: u64ToStrIfNonZero(m.validationFees),
    vs_operator: m.vsOperator || undefined,
    vs_operator_authz_msg_types: m.vsOperatorAuthzMsgTypes?.length ? m.vsOperatorAuthzMsgTypes : undefined,
    vs_operator_authz_spend_limit: m.vsOperatorAuthzSpendLimit?.length ? m.vsOperatorAuthzSpendLimit : null,
    vs_operator_authz_with_feegrant: m.vsOperatorAuthzWithFeegrant || undefined,
    vs_operator_authz_fee_spend_limit: m.vsOperatorAuthzFeeSpendLimit?.length ? m.vsOperatorAuthzFeeSpendLimit : null,
    vs_operator_authz_period: durationToAmino(m.vsOperatorAuthzPeriod),
  }),
  fromAmino: (a: any): MsgSelfCreateParticipant =>
    MsgSelfCreateParticipant.fromPartial({
      corporation: a.corporation ?? "",
      operator: a.operator ?? "",
      role: a.role ?? 0,
      validatorParticipantId: strToU64(a.validator_participant_id) != null ? Number(strToU64(a.validator_participant_id)!.toString()) : 0,
      did: a.did ?? "",
      effectiveFrom: isoToDate(a.effective_from),
      effectiveUntil: isoToDate(a.effective_until),
      verificationFees: strToU64(a.verification_fees) != null ? Number(strToU64(a.verification_fees)!.toString()) : 0,
      validationFees: strToU64(a.validation_fees) != null ? Number(strToU64(a.validation_fees)!.toString()) : 0,
      vsOperator: a.vs_operator ?? "",
      vsOperatorAuthzMsgTypes: a.vs_operator_authz_msg_types ?? [],
      vsOperatorAuthzSpendLimit: a.vs_operator_authz_spend_limit ?? [],
      vsOperatorAuthzWithFeegrant: a.vs_operator_authz_with_feegrant ?? false,
      vsOperatorAuthzFeeSpendLimit: a.vs_operator_authz_fee_spend_limit ?? [],
      vsOperatorAuthzPeriod: aminoToDuration(a.vs_operator_authz_period),
    }),
};
