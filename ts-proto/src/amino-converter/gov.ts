import type { AminoConverter, AminoTypes } from "@cosmjs/stargate";
import type { Registry } from "@cosmjs/proto-signing";
import { MsgSubmitProposal, MsgVote } from "cosmjs-types/cosmos/gov/v1/tx";
import { clean } from "./util/helpers";

// Amino converters for cosmos x/gov v1 messages (MsgSubmitProposal, MsgVote).
//
// Like the x/group equivalent, cosmjs does not recursively amino-encode the
// Any[] in MsgSubmitProposal.messages, so this factory transcodes each inner
// message through the AminoTypes registry (provided via closure). This is what
// lets gov-only verana messages (e.g. the XR grant/revoke authorizations) be
// amino-signed inside a governance proposal.
//
// Amino type names from cosmos-sdk x/gov v1 codec:
//   MsgSubmitProposal → "cosmos-sdk/v1/MsgSubmitProposal"
//   MsgVote           → "cosmos-sdk/v1/MsgVote"
export function createGovAminoConverters(
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
    "/cosmos.gov.v1.MsgSubmitProposal": {
      aminoType: "cosmos-sdk/v1/MsgSubmitProposal",
      toAmino: (msg: MsgSubmitProposal) =>
        clean({
          messages: msg.messages?.length ? msg.messages.map(encodeInner) : undefined,
          initial_deposit: msg.initialDeposit?.length
            ? msg.initialDeposit.map((c) => ({ denom: c.denom, amount: c.amount }))
            : undefined,
          proposer: msg.proposer || undefined,
          metadata: msg.metadata || undefined,
          title: msg.title || undefined,
          summary: msg.summary || undefined,
        }),
      fromAmino: (value: any): MsgSubmitProposal =>
        MsgSubmitProposal.fromPartial({
          messages: Array.isArray(value.messages) ? value.messages.map(decodeInner) : [],
          initialDeposit: Array.isArray(value.initial_deposit) ? value.initial_deposit : [],
          proposer: value.proposer ?? "",
          metadata: value.metadata ?? "",
          title: value.title ?? "",
          summary: value.summary ?? "",
        }),
    },

    "/cosmos.gov.v1.MsgVote": {
      aminoType: "cosmos-sdk/v1/MsgVote",
      toAmino: (msg: MsgVote) =>
        clean({
          proposal_id: msg.proposalId !== undefined ? String(msg.proposalId) : undefined,
          voter: msg.voter || undefined,
          option: msg.option || undefined,
          metadata: msg.metadata || undefined,
        }),
      fromAmino: (value: any): MsgVote =>
        MsgVote.fromPartial({
          proposalId: value.proposal_id != null ? BigInt(value.proposal_id) : BigInt(0),
          voter: value.voter ?? "",
          option: value.option ?? 0,
          metadata: value.metadata ?? "",
        }),
    },
  };
}
