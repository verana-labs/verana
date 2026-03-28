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
  toAmino: ({ authority, operator, did, aka, language, docUrl, docDigestSri }: MsgCreateTrustRegistry) => ({
    authority,
    operator,
    did,
    aka,
    language,
    doc_url: docUrl,
    doc_digest_sri: docDigestSri,
  }),
  fromAmino: (value: any) =>
    MsgCreateTrustRegistry.fromPartial({
      authority: value.authority,
      operator: value.operator,
      did: value.did,
      aka: value.aka,
      language: value.language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
    }),
};

export const MsgUpdateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgUpdateTrustRegistry",
  toAmino: ({ authority, operator, id, did, aka }: MsgUpdateTrustRegistry) => ({
    authority,
    operator,
    id: id != null ? id.toString() : undefined,
    did,
    aka,
  }),
  fromAmino: (value: any) =>
    MsgUpdateTrustRegistry.fromPartial({
      authority: value.authority,
      operator: value.operator,
      id: value.id != null ? Number(value.id) : 0,
      did: value.did,
      aka: value.aka,
    }),
};

export const MsgArchiveTrustRegistryAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgArchiveTrustRegistry",
  toAmino: ({ authority, operator, id, archive }: MsgArchiveTrustRegistry) => ({
    authority,
    operator,
    id: id != null ? id.toString() : undefined,
    archive,
  }),
  fromAmino: (value: any) =>
    MsgArchiveTrustRegistry.fromPartial({
      authority: value.authority,
      operator: value.operator,
      id: value.id != null ? Number(value.id) : 0,
      archive: value.archive,
    }),
};

export const MsgAddGovernanceFrameworkDocumentAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgAddGovernanceFrameworkDocument",
  toAmino: ({
    authority,
    operator,
    id,
    docLanguage,
    docUrl,
    docDigestSri,
    version,
  }: MsgAddGovernanceFrameworkDocument) => ({
    authority,
    operator,
    id: id != null ? id.toString() : undefined,
    doc_language: docLanguage,
    doc_url: docUrl,
    doc_digest_sri: docDigestSri,
    version,
  }),
  fromAmino: (value: any) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      authority: value.authority,
      operator: value.operator,
      id: value.id != null ? Number(value.id) : 0,
      docLanguage: value.doc_language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
      version: value.version,
    }),
};

export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: "/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion",
  toAmino: ({ authority, operator, id }: MsgIncreaseActiveGovernanceFrameworkVersion) => ({
    authority,
    operator,
    id: id != null ? id.toString() : undefined,
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      authority: value.authority,
      operator: value.operator,
      id: value.id != null ? Number(value.id) : 0,
    }),
};
