import { type AminoConverter } from '@cosmjs/stargate'
import {
  MsgReclaimTrustDepositYield,
  MsgReclaimTrustDeposit,
  MsgRepaySlashedTrustDeposit,
} from '../codec/verana/td/v1/tx'
import { strToU64, u64ToStr } from './util/helpers';

/**
 * Amino converter for MsgReclaimTrustDeposit
 */
export const MsgReclaimTrustDepositAminoConverter = {
  aminoType: '/verana.td.v1.MsgReclaimTrustDeposit',
  toAmino: (msg: MsgReclaimTrustDeposit) => ({
    creator: msg.creator,
    claimed: u64ToStr(msg.claimed) // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgReclaimTrustDeposit.fromPartial({
      creator: value.creator,
      claimed: strToU64(value.claimed.toString()) // string -> Long (uint64)
    }),
}

/**
 * Amino converter for MsgReclaimTrustDepositYield
 */
export const MsgReclaimTrustDepositYieldAminoConverter: AminoConverter = {
  aminoType: '/verana.td.v1.MsgReclaimTrustDepositYield',
  toAmino: (msg: MsgReclaimTrustDepositYield) => ({
    creator: msg.creator,
  }),
  fromAmino: (value: any) =>
    MsgReclaimTrustDepositYield.fromPartial({
      creator: value.creator,
    }),
}

/**
 * Amino converter for MsgRepaySlashedTrustDeposit
 */
export const MsgRepaySlashedTrustDepositAminoConverter: AminoConverter = {
  aminoType: '/verana.td.v1.MsgRepaySlashedTrustDeposit',
  toAmino: (msg: MsgRepaySlashedTrustDeposit) => ({
    creator: msg.creator,
    account: msg.account,
    amount: u64ToStr(msg.amount) // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgRepaySlashedTrustDeposit.fromPartial({
      creator: value.creator,
      account: value.account,
      amount: strToU64(value.amount) // string -> Long (uint64)
    }),
}
