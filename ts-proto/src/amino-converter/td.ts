import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgReclaimTrustDeposit,
  MsgReclaimTrustDepositYield,
  MsgRepaySlashedTrustDeposit,
  MsgSlashTrustDeposit,
} from "../codec/verana/td/v1/tx";

export const MsgReclaimTrustDepositAminoConverter: AminoConverter = {
  aminoType: "/verana.td.v1.MsgReclaimTrustDeposit",
  toAmino: ({ creator, claimed }: MsgReclaimTrustDeposit) => ({
    creator,
    claimed: claimed != null ? claimed.toString() : undefined,
  }),
  fromAmino: (value: any) =>
    MsgReclaimTrustDeposit.fromPartial({
      creator: value.creator,
      claimed: value.claimed != null ? Number(value.claimed) : 0,
    }),
};

export const MsgReclaimTrustDepositYieldAminoConverter: AminoConverter = {
  aminoType: "/verana.td.v1.MsgReclaimTrustDepositYield",
  toAmino: ({ authority, operator }: MsgReclaimTrustDepositYield) => ({
    authority,
    operator,
  }),
  fromAmino: (value: any) =>
    MsgReclaimTrustDepositYield.fromPartial({
      authority: value.authority,
      operator: value.operator,
    }),
};

export const MsgRepaySlashedTrustDepositAminoConverter: AminoConverter = {
  aminoType: "/verana.td.v1.MsgRepaySlashedTrustDeposit",
  toAmino: ({ authority, operator, amount }: MsgRepaySlashedTrustDeposit) => ({
    authority,
    operator,
    amount: amount != null ? amount.toString() : undefined,
  }),
  fromAmino: (value: any) =>
    MsgRepaySlashedTrustDeposit.fromPartial({
      authority: value.authority,
      operator: value.operator,
      amount: value.amount != null ? Number(value.amount) : 0,
    }),
};

export const MsgSlashTrustDepositAminoConverter: AminoConverter = {
  aminoType: "/verana.td.v1.MsgSlashTrustDeposit",
  toAmino: ({ authority, account, amount }: MsgSlashTrustDeposit) => ({
    authority,
    account,
    amount,
  }),
  fromAmino: (value: any) =>
    MsgSlashTrustDeposit.fromPartial({
      authority: value.authority,
      account: value.account,
      amount: value.amount,
    }),
};
