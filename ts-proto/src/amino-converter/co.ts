import type { AminoConverter } from "@cosmjs/stargate";
import { Any } from "../codec/google/protobuf/any";
import {
  Member,
  MsgCreateCorporation,
  MsgUpdateCorporation,
} from "../codec/verana/co/v1/tx";
import { clean } from "./util/helpers";

// Members serialize as plain objects in amino.
function memberToAmino(m: Member) {
  return clean({
    address: m.address || undefined,
    weight: m.weight || undefined,
    metadata: m.metadata || undefined,
  });
}

function memberFromAmino(a: any): Member {
  return Member.fromPartial({
    address: a.address ?? "",
    weight: a.weight ?? "",
    metadata: a.metadata ?? "",
  });
}

// decision_policy is a google.protobuf.Any wrapping a x/group decision policy.
// Amino representation: {"type": typeUrl, "value": <base64-encoded protobuf>}.
function anyToAmino(a: Any | undefined) {
  if (!a) return undefined;
  return clean({
    type: a.typeUrl || undefined,
    value: a.value && a.value.length > 0 ? Buffer.from(a.value).toString("base64") : undefined,
  });
}

function anyFromAmino(v: any): Any {
  if (!v) return Any.fromPartial({ typeUrl: "", value: new Uint8Array() });
  const bytes = v.value ? new Uint8Array(Buffer.from(v.value, "base64")) : new Uint8Array();
  return Any.fromPartial({ typeUrl: v.type ?? "", value: bytes });
}

export const MsgCreateCorporationAminoConverter: AminoConverter = {
  aminoType: "verana/x/co/MsgCreateCorporation",
  toAmino: (m: MsgCreateCorporation) => clean({
    signer: m.signer || undefined,
    members: m.members && m.members.length > 0 ? m.members.map(memberToAmino) : undefined,
    group_metadata: m.groupMetadata || undefined,
    group_policy_metadata: m.groupPolicyMetadata || undefined,
    decision_policy: anyToAmino(m.decisionPolicy),
    did: m.did || undefined,
    language: m.language || undefined,
    doc_url: m.docUrl || undefined,
    doc_digest_sri: m.docDigestSri || undefined,
  }),
  fromAmino: (a: any): MsgCreateCorporation =>
    MsgCreateCorporation.fromPartial({
      signer: a.signer ?? "",
      members: Array.isArray(a.members) ? a.members.map(memberFromAmino) : [],
      groupMetadata: a.group_metadata ?? "",
      groupPolicyMetadata: a.group_policy_metadata ?? "",
      decisionPolicy: anyFromAmino(a.decision_policy),
      did: a.did ?? "",
      language: a.language ?? "",
      docUrl: a.doc_url ?? "",
      docDigestSri: a.doc_digest_sri ?? "",
    }),
};

export const MsgUpdateCorporationAminoConverter: AminoConverter = {
  aminoType: "verana/x/co/MsgUpdateCorporation",
  toAmino: ({ corporation, operator, did }: MsgUpdateCorporation) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    did: did || undefined,
  }),
  fromAmino: (value: any) =>
    MsgUpdateCorporation.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      did: value.did ?? "",
    }),
};
