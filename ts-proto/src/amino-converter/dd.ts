'use client';

import { AminoConverter } from '@cosmjs/stargate'
import { MsgAddDID, MsgRenewDID, MsgTouchDID, MsgRemoveDID } from '../codec/verana/dd/v1/tx'

/**
 * Amino converter for MsgAddDID
 */
export const MsgAddDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgAddDID',
  toAmino: (msg: MsgAddDID) => ({
    creator: msg.creator,
    did: msg.did,
    years: msg.years,
  }),
  fromAmino: (value: any) =>
    MsgAddDID.fromPartial({
      creator: value.creator,
      did: value.did,
      years: value.years,
    }),
}

/**
 * Amino converter for MsgRenewDID
 */
export const MsgRenewDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgRenewDID',
  toAmino: (msg: MsgRenewDID) => ({
    creator: msg.creator,
    did: msg.did,
    years: msg.years,
  }),
  fromAmino: (value: any) =>
    MsgRenewDID.fromPartial({
      creator: value.creator,
      did: value.did,
      years: value.years,
    }),
}

/**
 * Amino converter for MsgTouchDID
 */
export const MsgTouchDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgTouchDID',
  toAmino: (msg: MsgTouchDID) => ({
    creator: msg.creator,
    did: msg.did,
  }),
  fromAmino: (value: any) =>
    MsgTouchDID.fromPartial({
      creator: value.creator,
      did: value.did,
    }),
}

/**
 * Amino converter for MsgRemoveDID
 */
export const MsgRemoveDIDAminoConverter: AminoConverter = {
  aminoType: '/verana.dd.v1.MsgRemoveDID',
  toAmino: (msg: MsgRemoveDID) => ({
    creator: msg.creator,
    did: msg.did,
  }),
  fromAmino: (value: any) =>
    MsgRemoveDID.fromPartial({
      creator: value.creator,
      did: value.did,
    }),
}
