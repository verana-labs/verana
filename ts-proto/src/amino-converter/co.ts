import type { AminoConverter } from "@cosmjs/stargate";
import { Any } from "../codec/google/protobuf/any";
import {
  Member,
  MsgCreateCorporation,
  MsgUpdateCorporation,
} from "../codec/verana/co/v1/tx";
import {
  ThresholdDecisionPolicy,
  PercentageDecisionPolicy,
} from "cosmjs-types/cosmos/group/v1/types";
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

// Convert a proto Duration ({ seconds: bigint, nanos: number }) to the amino
// nanosecond integer string that Go amino produces for time.Duration fields.
function durationToNanosStr(d: { seconds?: bigint; nanos?: number } | undefined): string {
  if (!d) return "0";
  const secs = d.seconds ?? BigInt(0);
  const nanos = d.nanos ?? 0;
  return String(secs * BigInt(1_000_000_000) + BigInt(nanos));
}

function nanosStrToDuration(ns: string | undefined): { seconds: bigint; nanos: number } {
  const n = BigInt(ns ?? "0");
  return { seconds: n / BigInt(1_000_000_000), nanos: Number(n % BigInt(1_000_000_000)) };
}

// decision_policy is a google.protobuf.Any wrapping a cosmos x/group decision
// policy. The Go amino codec encodes it as:
//   {"type": "cosmos-sdk/ThresholdDecisionPolicy", "value": { ... amino fields ... }}
// where Duration fields are encoded as nanosecond integer strings.
function anyToAmino(a: Any | undefined): any {
  if (!a) return undefined;

  if (a.typeUrl === "/cosmos.group.v1.ThresholdDecisionPolicy") {
    const dp = ThresholdDecisionPolicy.decode(a.value);
    return clean({
      type: "cosmos-sdk/ThresholdDecisionPolicy",
      value: clean({
        threshold: dp.threshold || undefined,
        windows: dp.windows
          ? {
              voting_period: durationToNanosStr(dp.windows.votingPeriod as any),
              min_execution_period: durationToNanosStr(dp.windows.minExecutionPeriod as any),
            }
          : undefined,
      }),
    });
  }

  if (a.typeUrl === "/cosmos.group.v1.PercentageDecisionPolicy") {
    const dp = PercentageDecisionPolicy.decode(a.value);
    return clean({
      type: "cosmos-sdk/PercentageDecisionPolicy",
      value: clean({
        percentage: dp.percentage || undefined,
        windows: dp.windows
          ? {
              voting_period: durationToNanosStr(dp.windows.votingPeriod as any),
              min_execution_period: durationToNanosStr(dp.windows.minExecutionPeriod as any),
            }
          : undefined,
      }),
    });
  }

  // Fallback for unknown Any types.
  return clean({
    type: a.typeUrl || undefined,
    value: a.value && a.value.length > 0 ? Buffer.from(a.value).toString("base64") : undefined,
  });
}

function anyFromAmino(v: any): Any {
  if (!v) return Any.fromPartial({ typeUrl: "", value: new Uint8Array() });

  if (v.type === "cosmos-sdk/ThresholdDecisionPolicy") {
    const val = v.value ?? {};
    const dp = ThresholdDecisionPolicy.fromPartial({
      threshold: val.threshold ?? "",
      windows: val.windows
        ? {
            votingPeriod: nanosStrToDuration(val.windows.voting_period) as any,
            minExecutionPeriod: nanosStrToDuration(val.windows.min_execution_period) as any,
          }
        : undefined,
    });
    return Any.fromPartial({
      typeUrl: "/cosmos.group.v1.ThresholdDecisionPolicy",
      value: ThresholdDecisionPolicy.encode(dp).finish(),
    });
  }

  if (v.type === "cosmos-sdk/PercentageDecisionPolicy") {
    const val = v.value ?? {};
    const dp = PercentageDecisionPolicy.fromPartial({
      percentage: val.percentage ?? "",
      windows: val.windows
        ? {
            votingPeriod: nanosStrToDuration(val.windows.voting_period) as any,
            minExecutionPeriod: nanosStrToDuration(val.windows.min_execution_period) as any,
          }
        : undefined,
    });
    return Any.fromPartial({
      typeUrl: "/cosmos.group.v1.PercentageDecisionPolicy",
      value: PercentageDecisionPolicy.encode(dp).finish(),
    });
  }

  // Fallback: treat value as base64-encoded proto bytes.
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
