/**
 * Amino Converters for Verana Message Types
 * 
 * These converters match the frontend implementation to ensure compatibility.
 * They convert between Proto messages and Amino JSON format.
 */

import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";
import { AminoConverter } from "@cosmjs/stargate";

/**
 * Amino converter for MsgCreateTrustRegistry
 * Matches frontend implementation in aminoConvertersTR.ts
 */
export const MsgCreateTrustRegistryAminoConverter: AminoConverter = {
  aminoType: '/verana.tr.v1.MsgCreateTrustRegistry',
  toAmino: ({
    creator,
    did,
    aka,
    language,
    docUrl,
    docDigestSri,
  }: MsgCreateTrustRegistry) => ({
    creator,
    did,
    aka,
    language,
    doc_url: docUrl,
    doc_digest_sri: docDigestSri,
  }),
  fromAmino: (value: {
    creator: string;
    did: string;
    aka: string;
    language: string;
    doc_url: string;
    doc_digest_sri: string;
  }) =>
    MsgCreateTrustRegistry.fromPartial({
      creator: value.creator,
      did: value.did,
      aka: value.aka,
      language: value.language,
      docUrl: value.doc_url,
      docDigestSri: value.doc_digest_sri,
    }),
};

