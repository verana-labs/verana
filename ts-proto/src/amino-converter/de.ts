import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgGrantOperatorAuthorization,
  MsgRevokeOperatorAuthorization,
} from "../codec/verana/de/v1/tx";
import {
  aminoToDuration,
  clean,
  dateToIsoAmino,
  durationToAmino,
  isoToDate,
} from "./util/helpers";

export const MsgGrantOperatorAuthorizationAminoConverter: AminoConverter = {
  aminoType: "/verana.de.v1.MsgGrantOperatorAuthorization",
  toAmino: (m: MsgGrantOperatorAuthorization) => clean({
    authority: m.authority || undefined,
    operator: m.operator || undefined,
    grantee: m.grantee || undefined,
    msg_types: m.msgTypes?.length ? m.msgTypes : undefined,
    expiration: dateToIsoAmino(m.expiration),
    authz_spend_limit: m.authzSpendLimit?.length ? m.authzSpendLimit : undefined,
    authz_spend_limit_period: durationToAmino(m.authzSpendLimitPeriod),
    with_feegrant: m.withFeegrant || undefined,
    feegrant_spend_limit: m.feegrantSpendLimit?.length ? m.feegrantSpendLimit : undefined,
    feegrant_spend_limit_period: durationToAmino(m.feegrantSpendLimitPeriod),
  }),
  fromAmino: (a: any): MsgGrantOperatorAuthorization =>
    MsgGrantOperatorAuthorization.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      grantee: a.grantee ?? "",
      msgTypes: a.msg_types ?? [],
      expiration: isoToDate(a.expiration),
      authzSpendLimit: a.authz_spend_limit ?? [],
      authzSpendLimitPeriod: aminoToDuration(a.authz_spend_limit_period),
      withFeegrant: a.with_feegrant ?? false,
      feegrantSpendLimit: a.feegrant_spend_limit ?? [],
      feegrantSpendLimitPeriod: aminoToDuration(a.feegrant_spend_limit_period),
    }),
};

export const MsgRevokeOperatorAuthorizationAminoConverter: AminoConverter = {
  aminoType: "/verana.de.v1.MsgRevokeOperatorAuthorization",
  toAmino: ({ authority, operator, grantee }: MsgRevokeOperatorAuthorization) => clean({
    authority: authority || undefined,
    operator: operator || undefined,
    grantee: grantee || undefined,
  }),
  fromAmino: (value: any) =>
    MsgRevokeOperatorAuthorization.fromPartial({
      authority: value.authority,
      operator: value.operator,
      grantee: value.grantee,
    }),
};
