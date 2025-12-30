import { Secp256k1HdWallet, makeSignDoc, serializeSignDoc } from "@cosmjs/amino";
import { Secp256k1, Secp256k1Signature, sha256 } from "@cosmjs/crypto";
import { fromBase64, toHex } from "@cosmjs/encoding";
import { StargateClient } from "@cosmjs/stargate";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { MsgCreatePermission } from "../../src/codec/verana/perm/v1/tx";
import { PermissionType } from "../../src/codec/verana/perm/v1/types";
import { MsgCreatePermissionAminoConverter } from "../src/helpers/aminoConverters";

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

function buildCreatePermissionMsgs(): { clientMsg: AminoMsg; serverMsg: AminoMsg } {
  const effectiveFrom = new Date("2025-01-01T00:00:00.123Z");
  const effectiveUntil = new Date("2025-12-31T00:00:00.123Z");
  const protoMsg = MsgCreatePermission.fromPartial({
    creator: "verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4",
    schemaId: 1,
    type: PermissionType.VERIFIER,
    did: "did:verana:test:bench",
    country: "US",
    effectiveFrom,
    effectiveUntil,
    verificationFees: 0,
    validationFees: 0,
  });

  const serverValue = MsgCreatePermissionAminoConverter.toAmino(protoMsg);
  const clientValue = {
    ...serverValue,
    verification_fees: "0",
    validation_fees: "0",
  };

  return {
    // Use legacy amino type string to match the Go bench output.
    serverMsg: { type: "/perm/v1/create-perm", value: serverValue },
    clientMsg: { type: "/perm/v1/create-perm", value: clientValue },
  };
}

async function main() {
  const wallet = await Secp256k1HdWallet.fromMnemonic(MNEMONIC, { prefix: "verana" });
  const [account] = await wallet.getAccounts();

  // Account number and sequence come from chain state for this address.
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

  const { clientMsg, serverMsg } = buildCreatePermissionMsgs();

  const accountNumberStr = String(accountNumber);
  const sequenceStr = String(sequence);

  // Client-style bytes (bad): includes zero fee fields.
  const signDocWithZeros = makeSignDoc(
    [clientMsg],
    FEE,
    CHAIN_ID,
    MEMO,
    accountNumberStr,
    sequenceStr
  );
  // Server-style bytes (good): omits zero fee fields via Amino omitempty rules.
  const signDocOmittingZeros = makeSignDoc(
    [serverMsg],
    FEE,
    CHAIN_ID,
    MEMO,
    accountNumberStr,
    sequenceStr
  );

  const signedWithZeros = await wallet.signAmino(account.address, signDocWithZeros);
  const signedOmittingZeros = await wallet.signAmino(account.address, signDocOmittingZeros);

  const pubkeyBase64 = Buffer.from(account.pubkey).toString("base64");

  const verifyZerosOnZeros = await Secp256k1.verifySignature(
    Secp256k1Signature.fromFixedLength(fromBase64(signedWithZeros.signature.signature)),
    sha256(serializeSignDoc(signDocWithZeros)),
    fromBase64(pubkeyBase64)
  );

  const verifyOmitOnOmit = await Secp256k1.verifySignature(
    Secp256k1Signature.fromFixedLength(fromBase64(signedOmittingZeros.signature.signature)),
    sha256(serializeSignDoc(signDocOmittingZeros)),
    fromBase64(pubkeyBase64)
  );

  const verifyZerosOnOmit = await Secp256k1.verifySignature(
    Secp256k1Signature.fromFixedLength(fromBase64(signedWithZeros.signature.signature)),
    sha256(serializeSignDoc(signDocOmittingZeros)),
    fromBase64(pubkeyBase64)
  );

  const verifyOmitOnZeros = await Secp256k1.verifySignature(
    Secp256k1Signature.fromFixedLength(fromBase64(signedOmittingZeros.signature.signature)),
    sha256(serializeSignDoc(signDocWithZeros)),
    fromBase64(pubkeyBase64)
  );

  console.log("Message with zero fees:");
  console.log(JSON.stringify(clientMsg, null, 2));
  console.log();

  console.log("Message omitting zero fees:");
  console.log(JSON.stringify(serverMsg, null, 2));
  console.log();

  console.log("Signature verifies against its own sign bytes:");
  console.log("  with zeros    ->", verifyZerosOnZeros);
  console.log("  omit zeros    ->", verifyOmitOnOmit);
  console.log();

  const bytesWithZeros = serializeSignDoc(signDocWithZeros);
  const bytesOmittingZeros = serializeSignDoc(signDocOmittingZeros);
  const scriptDir = dirname(fileURLToPath(import.meta.url));
  const outDir = join(scriptDir, "..", "out");
  mkdirSync(outDir, { recursive: true });
  writeFileSync(join(outDir, "amino-sign-bench-ts-server.json"), JSON.stringify(signDocOmittingZeros, null, 2));
  writeFileSync(join(outDir, "amino-sign-bench-ts-client.json"), JSON.stringify(signDocWithZeros, null, 2));
  writeFileSync(join(outDir, "amino-sign-bench-ts-server.hex"), `${toHex(bytesOmittingZeros)}\n`);
  writeFileSync(join(outDir, "amino-sign-bench-ts-client.hex"), `${toHex(bytesWithZeros)}\n`);
  console.log(`Wrote TS outputs to ${outDir}`);
  console.log("Server sign bytes (legacy Amino JSON, omitempty):");
  console.log(JSON.stringify(signDocOmittingZeros, null, 2));
  console.log();
  console.log("Client-style bytes (zeros included, different bytes):");
  console.log(JSON.stringify(signDocWithZeros, null, 2));
  console.log();
  console.log("Sign bytes hex (client-style, zeros included):");
  console.log();
  console.log(toHex(bytesWithZeros));
  console.log();
  console.log("Sign bytes hex (server-style, zeros omitted):");
  console.log();
  console.log(toHex(bytesOmittingZeros));
  console.log();

  console.log("Signature verifies against the OTHER sign bytes:");
  console.log("  zeros vs omit ->", verifyZerosOnOmit);
  console.log("  omit vs zeros ->", verifyOmitOnZeros);
  console.log();

  console.log("Why this breaks: if the chain omits zero fees but you sign with them,");
  console.log("your signature is over different bytes and verification fails.");
}

main().catch((err) => {
  console.error("bench error:", err.message || err);
  process.exit(1);
});
