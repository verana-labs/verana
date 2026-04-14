import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgAddGovernanceFrameworkDocument,
  MsgArchiveTrustRegistry,
  MsgCreateTrustRegistry,
  MsgIncreaseActiveGovernanceFrameworkVersion,
  MsgUpdateTrustRegistry,
} from "../codec/verana/tr/v1/tx";

export const MsgCreateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgCreateTrustRegistry",
  toAmino: ({ corporation, operator, did, aka, language }: MsgCreateTrustRegistry) => ({
    corporation,
    operator,
    did,
    aka,
    language,
  }),
  fromAmino: (value: any) =>
    MsgCreateTrustRegistry.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
      did: value.did,
      aka: value.aka,
      language: value.language,
    }),
};

export const MsgUpdateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgUpdateTrustRegistry",
  toAmino: ({ corporation, operator, trId, aka, language }: MsgUpdateTrustRegistry) => ({
    corporation,
    operator,
    tr_id: trId != null ? trId.toString() : undefined,
    aka,
    language,
  }),
  fromAmino: (value: any) =>
    MsgUpdateTrustRegistry.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
      aka: value.aka,
      language: value.language,
    }),
};

export const MsgArchiveTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgArchiveTrustRegistry",
  toAmino: ({ corporation, operator, trId, archive }: MsgArchiveTrustRegistry) => ({
    corporation,
    operator,
    tr_id: trId != null ? trId.toString() : undefined,
    archive,
  }),
  fromAmino: (value: any) =>
    MsgArchiveTrustRegistry.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
      archive: value.archive,
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
  }: MsgAddGovernanceFrameworkDocument) => ({
    corporation,
    operator,
    tr_id: trId != null ? trId.toString() : undefined,
    language,
    url,
    digest_sri: digestSri,
    version,
  }),
  fromAmino: (value: any) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
      language: value.language,
      url: value.url,
      digestSri: value.digest_sri,
      version: value.version,
    }),
};

export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
  toAmino: ({ corporation, operator, trId }: MsgIncreaseActiveGovernanceFrameworkVersion) => ({
    corporation,
    operator,
    tr_id: trId != null ? trId.toString() : undefined,
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      corporation: value.corporation,
      operator: value.operator,
      trId: value.tr_id != null ? Number(value.tr_id) : 0,
    }),
};
