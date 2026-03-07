import {
  MsgCreateTrustRegistry,
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
  MsgUpdateTrustRegistry,
  MsgArchiveTrustRegistry,
} from '../codec/verana/tr/v1/tx'
import { strToU64, u64ToStr } from './util/helpers';

/**
 * Amino converter for MsgCreateTrustRegistry
 */
export const MsgCreateTrustRegistryAminoConverter = {
  aminoType: '/verana.tr.v1.MsgCreateTrustRegistry',
  toAmino: (msg: MsgCreateTrustRegistry) => ({
    creator: msg.creator,
    did: msg.did,
    aka: msg.aka,
    language: msg.language,
    doc_url: msg.docUrl,
    doc_digest_sri: msg.docDigestSri,
  }),
  fromAmino: (value: any) =>
    MsgCreateTrustRegistry.fromPartial({
      creator: value.creator,
      did: value.did,
      aka: value.aka,
      language: value.language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
    }),
};

/**
 * Amino converter for MsgAddGovernanceFrameworkDocument
 */
export const MsgAddGovernanceFrameworkDocumentAminoConverter = {
  aminoType: '/verana.tr.v1.MsgAddGovernanceFrameworkDocument',
  toAmino: (msg: MsgAddGovernanceFrameworkDocument) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
    doc_language: msg.docLanguage,
    doc_url: msg.docUrl,
    doc_digest_sri: msg.docDigestSri,
    version: msg.version,
  }),
  fromAmino: (value: any) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
      docLanguage: value.doc_language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
      version: value.version,
    }),
};

/**
 * Amino converter for MsgIncreaseActiveGovernanceFrameworkVersion
 */
export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter = {
  aminoType: '/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion',
  toAmino: (msg: MsgIncreaseActiveGovernanceFrameworkVersion) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id) // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
    }),
};

/**
 * Amino converter for MsgUpdateTrustRegistry
 */
export const MsgUpdateTrustRegistryAminoConverter = {
  aminoType: '/verana.tr.v1.MsgUpdateTrustRegistry',
  toAmino: (msg: MsgUpdateTrustRegistry) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
    did: msg.did,
    aka: msg.aka,
  }),
  fromAmino: (value: any) =>
    MsgUpdateTrustRegistry.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
      did: value.did,
      aka: value.aka,
    }),
};

/**
 * Amino converter for MsgArchiveTrustRegistry
 */
export const MsgArchiveTrustRegistryAminoConverter = {
  aminoType: '/verana.tr.v1.MsgArchiveTrustRegistry',
  toAmino: (msg: MsgArchiveTrustRegistry) => ({
    creator: msg.creator,
    id: u64ToStr(msg.id), // uint64 -> string
    archive: msg.archive ? true : undefined, // omit if false
  }),
  fromAmino: (value: any) =>
    MsgArchiveTrustRegistry.fromPartial({
      creator: value.creator,
      id: strToU64(value.id), // string -> Long (uint64)
      archive: value.archive ?? false,
    }),
};
