import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgReclaimTrustDepositYield,
  MsgRepaySlashedTrustDeposit,
  MsgSlashTrustDeposit,
} from "../codec/verana/td/v1/tx";

export const MsgReclaimTrustDepositYieldAminoConverter: AminoConverter = {
  aminoType: "verana/x/td/MsgReclaimTrustDepositYield",
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
  toAmino: ({ authority, corporation, deposit }: MsgSlashTrustDeposit) => ({
    authority,
    corporation,
    deposit,
  }),
  fromAmino: (value: any) =>
    MsgSlashTrustDeposit.fromPartial({
      authority: value.authority,
      corporation: value.corporation,
      deposit: value.deposit,
    }),
};
