/**
 * Debug: Compare TS amino encoding with Go aminojson output
 */
import {
  MsgGrantOperatorAuthorizationAminoConverter,
  MsgCreateTrustRegistryAminoConverter,
} from "../../../src/helpers/aminoConverters";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { MsgCreateTrustRegistry } from "../../../src/codec/verana/tr/v1/tx";

function sortedStringify(obj: any): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

// Test 1: MsgGrantOperatorAuthorization (empty operator)
console.log("=== MsgGrantOperatorAuthorization (empty operator) ===");
const grantMsg = MsgGrantOperatorAuthorization.fromPartial({
  authority: "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
  operator: "",
  grantee: "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
  msgTypes: [
    "/verana.tr.v1.MsgCreateTrustRegistry",
    "/verana.tr.v1.MsgUpdateTrustRegistry",
  ],
});

const aminoGrant = MsgGrantOperatorAuthorizationAminoConverter.toAmino(grantMsg);
console.log("TS amino type:", MsgGrantOperatorAuthorizationAminoConverter.aminoType);
console.log("TS amino value:", JSON.stringify(aminoGrant));
console.log("TS amino value (sorted):", sortedStringify(aminoGrant));

const goGrant = `{"authority":"verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt","grantee":"verana13627hsukut3p39l5lawa88er0az36227rkwt83","msg_types":["/verana.tr.v1.MsgCreateTrustRegistry","/verana.tr.v1.MsgUpdateTrustRegistry"]}`;
console.log("Go aminojson:  ", goGrant);
console.log("MATCH:", sortedStringify(aminoGrant) === goGrant ? "YES ✅" : "NO ❌");

// Test 2: MsgCreateTrustRegistry
console.log("\n=== MsgCreateTrustRegistry ===");
const createMsg = MsgCreateTrustRegistry.fromPartial({
  authority: "verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt",
  operator: "verana13627hsukut3p39l5lawa88er0az36227rkwt83",
  did: "did:verana:test:1234567890:abc123",
  aka: "http://ts-proto-test-trust-registry.com",
  language: "en",
  docUrl: "https://example.com/governance-framework.pdf",
  docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
});

const aminoCreate = MsgCreateTrustRegistryAminoConverter.toAmino(createMsg);
console.log("TS amino type:", MsgCreateTrustRegistryAminoConverter.aminoType);
console.log("TS amino value:", JSON.stringify(aminoCreate));
console.log("TS amino value (sorted):", sortedStringify(aminoCreate));

const goCreate = `{"aka":"http://ts-proto-test-trust-registry.com","authority":"verana1f06a8j6n02ash0vtkdge0a9pvvfa5ypghkyttt","did":"did:verana:test:1234567890:abc123","doc_digest_sri":"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26","doc_url":"https://example.com/governance-framework.pdf","language":"en","operator":"verana13627hsukut3p39l5lawa88er0az36227rkwt83"}`;
console.log("Go aminojson:  ", goCreate);
console.log("MATCH:", sortedStringify(aminoCreate) === goCreate ? "YES ✅" : "NO ❌");
