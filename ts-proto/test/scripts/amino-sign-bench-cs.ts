import { Secp256k1HdWallet, makeSignDoc, serializeSignDoc } from "@cosmjs/amino";
import { Secp256k1, Secp256k1Signature, sha256 } from "@cosmjs/crypto";
import { fromBase64, toHex } from "@cosmjs/encoding";
import { StargateClient } from "@cosmjs/stargate";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import Long from "long";
import { MsgCreateCredentialSchema, OptionalUInt32 } from "../../src/codec/verana/cs/v1/tx";
import { CredentialSchemaPermManagementMode } from "../../src/codec/verana/cs/v1/types";
import { MsgCreateCredentialSchemaAminoConverter } from "../src/helpers/aminoConverters";

type AminoMsg = {
  type: string;
  value: Record<string, unknown>;
};

const MNEMONIC =
  process.env.MNEMONIC ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const CHAIN_ID = process.env.VERANA_CHAIN_ID || "vna-testnet-1";
const RPC_ENDPOINT = process.env.VERANA_RPC_ENDPOINT || "http://localhost:26657";

const FEE = {
  amount: [{ amount: "557532", denom: "uvna" }],
  gas: "185844",
};

const MEMO = "Amino bench demo";

const JSON_SCHEMA =
  "{\"$id\":\"vpr:verana:VPR_CHAIN_ID/cs/v1/js/VPR_CREDENTIAL_SCHEMA_ID\",\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"title\":\"ExampleCredential\",\"description\":\"ExampleCredential using JsonSchema\",\"type\":\"object\",\"properties\":{\"credentialSubject\":{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"string\",\"format\":\"uri\"},\"firstName\":{\"type\":\"string\",\"minLength\":0,\"maxLength\":256},\"lastName\":{\"type\":\"string\",\"minLength\":1,\"maxLength\":256},\"expirationDate\":{\"type\":\"string\",\"format\":\"date\"},\"countryOfResidence\":{\"type\":\"string\",\"minLength\":2,\"maxLength\":2}},\"required\":[\"id\",\"lastName\",\"expirationDate\",\"countryOfResidence\"]}}}";

function buildCreateCredentialSchemaMsg(): AminoMsg {
  const protoMsg = MsgCreateCredentialSchema.fromPartial({
    creator: "verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4",
    trId: Long.fromNumber(1),
    jsonSchema: JSON_SCHEMA,
    issuerGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
    verifierGrantorValidationValidityPeriod: { value: 0 } as OptionalUInt32,
    issuerValidationValidityPeriod: { value: 0 } as OptionalUInt32,
    verifierValidationValidityPeriod: { value: 180 } as OptionalUInt32,
    holderValidationValidityPeriod: { value: 0 } as OptionalUInt32,
    issuerPermManagementMode: CredentialSchemaPermManagementMode.GRANTOR_VALIDATION,
    verifierPermManagementMode: CredentialSchemaPermManagementMode.OPEN,
  });

  return {
    // Legacy amino type string for CS.
    type: "/vpr/v1/cs/create-credential-schema",
    value: MsgCreateCredentialSchemaAminoConverter.toAmino(protoMsg),
  };
}

async function main() {
  const wallet = await Secp256k1HdWallet.fromMnemonic(MNEMONIC, { prefix: "verana" });
  const [account] = await wallet.getAccounts();

  let accountNumber = 0;
  let sequence = 111;
  try {
    const queryClient = await StargateClient.connect(RPC_ENDPOINT);
    ({ accountNumber, sequence } = await queryClient.getSequence(account.address));
    queryClient.disconnect();
    console.log(
      `Connected to ${RPC_ENDPOINT}. Using on-chain account_number=${accountNumber}, sequence=${sequence}.`
    );
  } catch (error: any) {
    console.log(
      `Could not connect to ${RPC_ENDPOINT}. Using defaults account_number=0, sequence=111.`
    );
  }

  const msg = buildCreateCredentialSchemaMsg();
  const signDoc = makeSignDoc(
    [msg],
    FEE,
    CHAIN_ID,
    MEMO,
    String(accountNumber),
    String(sequence)
  );

  const signed = await wallet.signAmino(account.address, signDoc);
  const pubkeyBase64 = Buffer.from(account.pubkey).toString("base64");
  const verify = await Secp256k1.verifySignature(
    Secp256k1Signature.fromFixedLength(fromBase64(signed.signature.signature)),
    sha256(serializeSignDoc(signDoc)),
    fromBase64(pubkeyBase64)
  );

  console.log("Credential Schema message:");
  console.log(JSON.stringify(msg, null, 2));
  console.log();
  console.log("Signature verifies against sign bytes:");
  console.log("  ok ->", verify);
  console.log();

  const bytes = serializeSignDoc(signDoc);
  const scriptDir = dirname(fileURLToPath(import.meta.url));
  const outDir = join(scriptDir, "..", "out");
  mkdirSync(outDir, { recursive: true });
  writeFileSync(join(outDir, "amino-sign-bench-cs-ts.json"), JSON.stringify(signDoc, null, 2));
  writeFileSync(join(outDir, "amino-sign-bench-cs-ts.hex"), `${toHex(bytes)}\n`);
  console.log(`Wrote TS outputs to ${outDir}`);
  console.log("Client sign bytes (legacy Amino JSON):");
  console.log(JSON.stringify(signDoc, null, 2));
  console.log();
  console.log("Sign bytes hex:");
  console.log();
  console.log(toHex(bytes));
}

main().catch((err) => {
  console.error("bench error:", err.message || err);
  process.exit(1);
});
