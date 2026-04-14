import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgAddGovernanceFrameworkDocument,
  MsgArchiveTrustRegistry,
  MsgCreateTrustRegistry,
  MsgIncreaseActiveGovernanceFrameworkVersion,
  MsgUpdateTrustRegistry,
} from "../codec/verana/tr/v1/tx";
import { clean, u64ToStr } from "./util/helpers";

export const MsgCreateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgCreateTrustRegistry",
  toAmino: ({ corporation, operator, did, aka, language }: MsgCreateTrustRegistry) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    did: did || undefined,
    aka: aka || undefined,
    language: language || undefined,
  }),
  fromAmino: (value: any) =>
    MsgCreateTrustRegistry.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      did: value.did ?? "",
      aka: value.aka ?? "",
      language: value.language ?? "",
    }),
};

export const MsgUpdateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgUpdateTrustRegistry",
  toAmino: ({ corporation, operator, trId, aka, language }: MsgUpdateTrustRegistry) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    tr_id: u64ToStr(trId as any),
    aka: aka || undefined,
    language: language || undefined,
  }),
  fromAmino: (value: any) =>
    MsgUpdateTrustRegistry.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
      aka: value.aka ?? "",
      language: value.language ?? "",
    }),
};

export const MsgArchiveTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgArchiveTrustRegistry",
  toAmino: ({ corporation, operator, trId, archive }: MsgArchiveTrustRegistry) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    tr_id: u64ToStr(trId as any),
    archive: archive || undefined,
  }),
  fromAmino: (value: any) =>
    MsgArchiveTrustRegistry.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
      archive: value.archive ?? false,
    }),
};

export const MsgAddGovernanceFrameworkDocumentAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgAddGovernanceFrameworkDocument",
  toAmino: ({
    corporation,
    operator,
    trId,
    language,
    url,
    digestSri,
    version,
  }: MsgAddGovernanceFrameworkDocument) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    tr_id: u64ToStr(trId as any),
    language: language || undefined,
    url: url || undefined,
    digest_sri: digestSri || undefined,
    version: version || undefined,
  }),
  fromAmino: (value: any) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
      language: value.language ?? "",
      url: value.url ?? "",
      digestSri: value.digest_sri ?? "",
      version: value.version ?? 0,
    }),
};

export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
  toAmino: ({ corporation, operator, trId }: MsgIncreaseActiveGovernanceFrameworkVersion) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    tr_id: u64ToStr(trId as any),
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
    }),
};
