/**
 * Journey: CO Create Corporation
 *
 * Atomically creates an x/group group + group policy AND registers the
 * resulting policy_address as a MOD-CO Corporation. The returned
 * `policy_address` is what AUTHZ-CHECK-5 accepts as the signing `corporation`
 * for downstream EC/GF/CS/PERM messages.
 *
 * This is the FIRST setup step in the v4-rc2 test suite (replaces the
 * standalone `x/group MsgCreateGroupWithPolicy` step that previously seeded TR).
 *
 * Usage:
 *   npm run test:co-create
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  createQueryClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
  generateUniqueDID,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgCreateCorporation } from "../../../src/codec/verana/co/v1/tx";
import { Any } from "../../../src/codec/google/protobuf/any";
import { ThresholdDecisionPolicy } from "cosmjs-types/cosmos/group/v1/types";
import { saveActiveCorporation } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// MOD-CO signer (the account that submits MsgCreateCorporation). After the tx,
// it holds NO ongoing admin privileges — the group_policy_address becomes the
// admin. Index 10 mirrors the legacy "authority" account used by the old TR
// flow, so existing per-test funding patterns continue to work.
const SIGNER_INDEX = 10;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: CO Create Corporation");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create signer wallet and fund it
  console.log("Step 1: Creating signer wallet...");
  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);
  const signerAccount = await getAccountInfo(signerWallet);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  console.log(`  Cooluser: ${cooluserAccount.address}`);
  console.log(`  Signer:   ${signerAccount.address} (derivation index ${SIGNER_INDEX})`);
  console.log();

  console.log("Step 2: Funding signer...");
  const fundAmount = "50000000uvna"; // 50 VNA
  const fundResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    signerAccount.address,
    fundAmount,
  );
  if (fundResult.code !== 0) {
    console.log(`  ❌ Failed to fund signer: ${fundResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  ✓ Funded signer with ${fundAmount}`);

  const queryClient = await createQueryClient();
  console.log("  ⏳ Waiting for funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundResult.transactionHash);
      if (tx) {
        console.log(`  ✓ Funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }
  for (let i = 0; i < 20; i++) {
    const balance = await queryClient.getBalance(signerAccount.address, config.denom);
    if (BigInt(balance.amount) > 0) {
      console.log(`  ✓ Signer balance: ${balance.amount}${balance.denom}`);
      break;
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  queryClient.disconnect();
  console.log();

  // Step 3: Build MsgCreateCorporation
  //
  // Members list: just the signer with weight "1" (single-member threshold "1"
  // group, voting period 1s — adequate for end-to-end transactional tests).
  console.log("Step 3: Building MsgCreateCorporation...");
  const did = generateUniqueDID();
  // Use cosmjs-types ThresholdDecisionPolicy directly (transitive dep via
  // @cosmjs/stargate). The chain decodes `decision_policy.value` against its
  // interface registry; using the canonical cosmos-sdk proto descriptor
  // guarantees byte-for-byte match.
  const decisionPolicyBytes = ThresholdDecisionPolicy.encode(
    ThresholdDecisionPolicy.fromPartial({
      threshold: "1",
      windows: {
        votingPeriod: { seconds: BigInt(30), nanos: 0 },
        minExecutionPeriod: { seconds: BigInt(0), nanos: 0 },
      },
    }),
  ).finish();

  const msg = {
    typeUrl: typeUrls.MsgCreateCorporation,
    value: MsgCreateCorporation.fromPartial({
      signer: signerAccount.address,
      members: [
        {
          address: signerAccount.address,
          weight: "1",
          metadata: "member_1",
        },
      ],
      groupMetadata: "ts-client-test corporation",
      groupPolicyMetadata: "threshold policy",
      decisionPolicy: Any.fromPartial({
        typeUrl: "/cosmos.group.v1.ThresholdDecisionPolicy",
        value: decisionPolicyBytes,
      }),
      did,
      language: "en",
      docUrl: "http://ts-proto-test-corporation.com/cgf-v1",
      docDigestSri: "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
    }),
  };

  console.log(`  Signer: ${signerAccount.address}`);
  console.log(`  DID:    ${did}`);
  console.log();

  // Step 4: Broadcast
  const client = await createSigningClient(signerWallet);
  try {
    const fee = await calculateFeeWithSimulation(
      client, signerAccount.address, [msg],
      "Creating Corporation",
    );
    console.log(`  Gas: ${fee.gas}, Fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client, signerAccount.address, [msg], fee,
      "Creating Corporation",
    );

    if (result.code === 0) {
      console.log();
      console.log("✅ SUCCESS! Corporation created!");
      console.log(`  Tx Hash: ${result.transactionHash}`);
      console.log(`  Block:   ${result.height}`);
      console.log(`  Gas:     ${result.gasUsed}/${result.gasWanted}`);

      let corporationId: number | undefined;
      let policyAddress: string | undefined;
      for (const event of (result.events || [])) {
        if (event.type === "create_corporation" || event.type === "verana.co.v1.EventCreateCorporation") {
          for (const attr of event.attributes) {
            if (attr.key === "corporation_id") {
              const v = parseInt(attr.value, 10);
              if (!isNaN(v)) corporationId = v;
            } else if (attr.key === "policy_address") {
              policyAddress = attr.value;
            }
          }
        }
      }

      if (corporationId && policyAddress) {
        console.log(`  Corp ID:        ${corporationId}`);
        console.log(`  Policy Address: ${policyAddress}`);
        saveActiveCorporation(corporationId, policyAddress);
        console.log("  💾 Saved active corporation for subsequent journeys");

        // Fund the group policy address so it can act as inner message sender
        // in x/group proposal execution (required by DE grant flow).
        console.log("\n  Funding policy address with 50 VNA...");
        const fundPolicyResult = await fundAccount(
          COOLUSER_MNEMONIC,
          cooluserAccount.address,
          policyAddress,
          "50000000uvna",
        );
        if (fundPolicyResult.code !== 0) {
          console.log(`  ⚠️  Policy address funding failed: ${fundPolicyResult.rawLog}`);
        } else {
          console.log(`  ✓ Policy address funding tx: ${fundPolicyResult.transactionHash}`);
          const pqc = await createQueryClient();
          for (let i = 0; i < 30; i++) {
            try {
              const tx = await pqc.getTx(fundPolicyResult.transactionHash);
              if (tx) { console.log(`  ✓ Policy address funded at block ${tx.height}`); break; }
            } catch {}
            await new Promise((r) => setTimeout(r, 1000));
          }
          pqc.disconnect();
        }
      } else {
        console.log("  ⚠️  Could not extract corporation_id/policy_address from events");
        process.exit(1);
      }
    } else {
      console.log("❌ FAILED!");
      console.log(`  Code: ${result.code}`);
      console.log(`  Log:  ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR!");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\n❌ Fatal error:", error.message || error);
  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
  }
  process.exit(1);
});
