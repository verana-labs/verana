import type { AminoConverter, AminoTypes } from "@cosmjs/stargate";
import type { Registry } from "@cosmjs/proto-signing";
import { MsgSubmitProposal, MsgVote } from "cosmjs-types/cosmos/group/v1/tx";
import { clean } from "./util/helpers";

// Amino converters for cosmos x/group messages (MsgSubmitProposal, MsgVote).
//
// CosmJS 0.32.x ships createGroupAminoConverters() as an empty {} because
// MsgSubmitProposal.messages is an Any[] that requires recursive amino
// encoding — the converter needs the AminoTypes registry to transcode each
// inner message. This factory receives that registry via closure at
// construction time (see createVeranaAminoTypes in signing.ts).
//
// Amino type names from cosmos-sdk x/group codec.go:
//   MsgSubmitProposal → "cosmos-sdk/group/MsgSubmitProposal"
//   MsgVote           → "cosmos-sdk/group/MsgVote"
//
// Field encoding notes (verified with go amino.MarshalJSON):
//   - exec: integer; absent when 0 (EXEC_UNSPECIFIED)
//   - option: integer (VoteOption enum)
//   - proposal_id: string (uint64 → string)
//   - messages[].type: amino type of the inner message
//   - messages[].value: amino fields of the inner message (recursively encoded)

export function createGroupAminoConverters(
  getAminoTypes: () => AminoTypes,
  registry: Registry,
): Record<string, AminoConverter> {
  const encodeInner = (any: { typeUrl: string; value: Uint8Array }) => {
    const genType = registry.lookupType(any.typeUrl);
    const decoded = genType ? genType.decode(any.value) : any;
    return getAminoTypes().toAmino({ typeUrl: any.typeUrl, value: decoded });
  };

  const decodeInner = (aminoMsg: any): { typeUrl: string; value: Uint8Array } => {
    const protoMsg = getAminoTypes().fromAmino(aminoMsg);
    const genType = registry.lookupType(protoMsg.typeUrl);
    const encoded = genType
      ? (genType as any).encode(protoMsg.value).finish()
      : new Uint8Array();
    return { typeUrl: protoMsg.typeUrl, value: encoded };
  };

  return {
    "/cosmos.group.v1.MsgSubmitProposal": {
      aminoType: "cosmos-sdk/group/MsgSubmitProposal",
      toAmino: (msg: MsgSubmitProposal) =>
        clean({
          group_policy_address: msg.groupPolicyAddress || undefined,
          proposers: msg.proposers?.length ? msg.proposers : undefined,
          metadata: msg.metadata || undefined,
          title: msg.title || undefined,
          summary: msg.summary || undefined,
          messages: msg.messages?.length
            ? msg.messages.map(encodeInner)
            : undefined,
          exec: msg.exec || undefined,
        }),
      fromAmino: (value: any): MsgSubmitProposal =>
        MsgSubmitProposal.fromPartial({
          groupPolicyAddress: value.group_policy_address ?? "",
          proposers: Array.isArray(value.proposers) ? value.proposers : [],
          metadata: value.metadata ?? "",
          title: value.title ?? "",
          summary: value.summary ?? "",
          messages: Array.isArray(value.messages)
            ? value.messages.map(decodeInner)
            : [],
          exec: value.exec ?? 0,
        }),
    },

    "/cosmos.group.v1.MsgVote": {
      aminoType: "cosmos-sdk/group/MsgVote",
      toAmino: (msg: MsgVote) =>
        clean({
          proposal_id:
            msg.proposalId !== undefined ? String(msg.proposalId) : undefined,
          voter: msg.voter || undefined,
          option: msg.option || undefined,
          metadata: msg.metadata || undefined,
          exec: msg.exec || undefined,
        }),
      fromAmino: (value: any): MsgVote =>
        MsgVote.fromPartial({
          proposalId:
            value.proposal_id != null ? BigInt(value.proposal_id) : BigInt(0),
          voter: value.voter ?? "",
          option: value.option ?? 0,
          metadata: value.metadata ?? "",
          exec: value.exec ?? 0,
        }),
    },
  };
}
