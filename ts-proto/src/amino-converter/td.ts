import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgReclaimTrustDepositYield,
  MsgRepaySlashedTrustDeposit,
  MsgSlashTrustDeposit,
} from "../codec/verana/td/v1/tx";
import { strToU64, u64ToStrIfNonZero } from "./util/helpers";

export const MsgReclaimTrustDepositYieldAminoConverter: AminoConverter = {
  aminoType: "verana/x/td/MsgReclaimTrustDepositYield",
  // [MOD-TD-MSG-2-1] Spec v4 draft 13: parameters are corporation + operator only.
  toAmino: ({ corporation, operator }: MsgReclaimTrustDepositYield) => ({
    corporation,
    operator,
  }),
  fromAmino: (value: any) =>
    MsgReclaimTrustDepositYield.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
    }),
};

export const MsgRepaySlashedTrustDepositAminoConverter: AminoConverter = {
  aminoType: "verana/x/td/MsgRepaySlashedTrustDeposit",
  toAmino: ({ corporation, operator, deposit }: MsgRepaySlashedTrustDeposit) => ({
    corporation,
    operator,
    deposit: deposit != null ? deposit.toString() : undefined,
  }),
  fromAmino: (value: any) =>
    MsgRepaySlashedTrustDeposit.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
      deposit: value.deposit != null ? Number(value.deposit) : 0,
    }),
};

export const MsgSlashTrustDepositAminoConverter: AminoConverter = {
  aminoType: "verana/x/td/MsgSlashTrustDeposit",
  // [MOD-TD-MSG-5] v4-rc3: target corporation_id (uint64); reason mandatory.
  toAmino: ({ authority, corporationId, deposit, reason }: MsgSlashTrustDeposit) => ({
    authority,
    corporation_id: u64ToStrIfNonZero(corporationId),
    deposit,
    reason,
  }),
  fromAmino: (value: any) =>
    MsgSlashTrustDeposit.fromPartial({
      authority: value.authority,
      corporationId: strToU64(value.corporation_id) != null ? Number(strToU64(value.corporation_id)!.toString()) : 0,
      deposit: value.deposit,
      reason: value.reason,
    }),
};
