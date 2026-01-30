/**
 * Verana Client Helper
 * Provides utilities for connecting to the Verana blockchain.
 */

import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Secp256k1HdWallet } from "@cosmjs/amino";
import { SigningStargateClient, StargateClient, GasPrice, calculateFee, AminoTypes } from "@cosmjs/stargate";
import { stringToPath } from "@cosmjs/crypto";
import { createVeranaRegistry } from "./registry";
import {
  // TR module
  MsgCreateTrustRegistryAminoConverter,
  MsgUpdateTrustRegistryAminoConverter,
  MsgArchiveTrustRegistryAminoConverter,
  MsgAddGovernanceFrameworkDocumentAminoConverter,
  MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
  // DD module
  MsgAddDIDAminoConverter,
  MsgRenewDIDAminoConverter,
  MsgTouchDIDAminoConverter,
  MsgRemoveDIDAminoConverter,
  // CS module
  MsgCreateCredentialSchemaAminoConverter,
  MsgUpdateCredentialSchemaAminoConverter,
  MsgArchiveCredentialSchemaAminoConverter,
  // TD module
  MsgReclaimTrustDepositAminoConverter,
  MsgReclaimTrustDepositYieldAminoConverter,
  // PERM module
  MsgCreateRootPermissionAminoConverter,
  MsgCreatePermissionAminoConverter,
  MsgExtendPermissionAminoConverter,
  MsgRevokePermissionAminoConverter,
  MsgStartPermissionVPAminoConverter,
  MsgRenewPermissionVPAminoConverter,
  MsgSetPermissionVPToValidatedAminoConverter,
  MsgCancelPermissionVPLastRequestAminoConverter,
  MsgCreateOrUpdatePermissionSessionAminoConverter,
} from "./aminoConverters";

// Default configuration - can be overridden via environment variables
// Matches frontend configuration from veranaChain.sign.client.ts
// Helper function to safely get env var with fallback
function getEnvOrDefault(key: string, defaultValue: string): string {
  const value = process.env[key];
  // Handle empty strings, null, undefined - treat as missing
  if (!value || typeof value !== 'string' || !value.trim()) {
    return defaultValue;
  }
  return value.trim();
}

// Create config as a getter function to ensure it reads env vars at access time
// This prevents issues where env vars are set after module load
function getConfig() {
  return {
    rpcEndpoint: getEnvOrDefault("VERANA_RPC_ENDPOINT", "http://localhost:26657"),
    lcdEndpoint: getEnvOrDefault("VERANA_LCD_ENDPOINT", "http://localhost:1317"),
    chainId: getEnvOrDefault("VERANA_CHAIN_ID", "verana"),
    addressPrefix: getEnvOrDefault("VERANA_ADDRESS_PREFIX", "verana"),
    denom: getEnvOrDefault("VERANA_DENOM", "uvna"),
    gasPrice: getEnvOrDefault("VERANA_GAS_PRICE", "3uvna"), // Matches frontend
    gasLimit: getEnvOrDefault("VERANA_GAS_LIMIT", "300000"), // Matches frontend
    gasAdjustment: parseFloat(getEnvOrDefault("VERANA_GAS_ADJUSTMENT", "2")), // Matches frontend
  };
}

// Export config as a getter to ensure fresh reads
export const config = new Proxy({} as ReturnType<typeof getConfig>, {
  get(_target, prop) {
    return getConfig()[prop as keyof ReturnType<typeof getConfig>];
  }
});

/**
 * Creates an Amino Sign wallet from a mnemonic phrase.
 * This matches the frontend's Amino Sign approach.
 */
export async function createAminoWallet(mnemonic: string): Promise<Secp256k1HdWallet> {
  return Secp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: config.addressPrefix,
  });
}

/**
 * Creates a Direct Sign wallet from a mnemonic phrase.
 * Kept for backward compatibility.
 */
export async function createDirectWallet(mnemonic: string): Promise<DirectSecp256k1HdWallet> {
  return DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: config.addressPrefix,
  });
}

/**
 * Creates a wallet from a mnemonic phrase.
 * Defaults to Amino Sign to match frontend behavior.
 */
export async function createWallet(mnemonic: string): Promise<Secp256k1HdWallet> {
  return createAminoWallet(mnemonic);
}

/**
 * Creates a wallet from a mnemonic phrase with a custom derivation path.
 * Used for creating multiple accounts from the same mnemonic.
 * @param mnemonic - Master mnemonic phrase
 * @param accountIndex - Account index for derivation path (e.g., 13, 14, 17, 21)
 * @returns Promise<Secp256k1HdWallet> for Amino signing
 */
export async function createAccountFromMnemonic(
  mnemonic: string,
  accountIndex: number
): Promise<Secp256k1HdWallet> {
  // Derivation path: m/44'/118'/0'/0/{accountIndex}
  const hdPath = stringToPath(`m/44'/118'/0'/0/${accountIndex}`);
  return Secp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: config.addressPrefix,
    hdPaths: [hdPath],
  });
}

/**
 * Creates Amino Types for Verana messages.
 * Matches frontend implementation in veranaChain.sign.client.ts
 */
export function createVeranaAminoTypes(): AminoTypes {
  return new AminoTypes({
    // Trust Registry (tr) module
    '/verana.tr.v1.MsgCreateTrustRegistry': MsgCreateTrustRegistryAminoConverter,
    '/verana.tr.v1.MsgUpdateTrustRegistry': MsgUpdateTrustRegistryAminoConverter,
    '/verana.tr.v1.MsgArchiveTrustRegistry': MsgArchiveTrustRegistryAminoConverter,
    '/verana.tr.v1.MsgAddGovernanceFrameworkDocument': MsgAddGovernanceFrameworkDocumentAminoConverter,
    '/verana.tr.v1.MsgIncreaseActiveGovernanceFrameworkVersion': MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter,
    // DID Directory (dd) module
    '/verana.dd.v1.MsgAddDID': MsgAddDIDAminoConverter,
    '/verana.dd.v1.MsgRenewDID': MsgRenewDIDAminoConverter,
    '/verana.dd.v1.MsgTouchDID': MsgTouchDIDAminoConverter,
    '/verana.dd.v1.MsgRemoveDID': MsgRemoveDIDAminoConverter,
    // Credential Schema (cs) module
    '/verana.cs.v1.MsgCreateCredentialSchema': MsgCreateCredentialSchemaAminoConverter,
    '/verana.cs.v1.MsgUpdateCredentialSchema': MsgUpdateCredentialSchemaAminoConverter,
    '/verana.cs.v1.MsgArchiveCredentialSchema': MsgArchiveCredentialSchemaAminoConverter,
    // Trust Deposit (td) module
    '/verana.td.v1.MsgReclaimTrustDeposit': MsgReclaimTrustDepositAminoConverter,
    '/verana.td.v1.MsgReclaimTrustDepositYield': MsgReclaimTrustDepositYieldAminoConverter,
    // Permission (perm) module
    '/verana.perm.v1.MsgCreateRootPermission': MsgCreateRootPermissionAminoConverter,
    '/verana.perm.v1.MsgCreatePermission': MsgCreatePermissionAminoConverter,
    '/verana.perm.v1.MsgExtendPermission': MsgExtendPermissionAminoConverter,
    '/verana.perm.v1.MsgRevokePermission': MsgRevokePermissionAminoConverter,
    '/verana.perm.v1.MsgStartPermissionVP': MsgStartPermissionVPAminoConverter,
    '/verana.perm.v1.MsgRenewPermissionVP': MsgRenewPermissionVPAminoConverter,
    '/verana.perm.v1.MsgSetPermissionVPToValidated': MsgSetPermissionVPToValidatedAminoConverter,
    '/verana.perm.v1.MsgCancelPermissionVPLastRequest': MsgCancelPermissionVPLastRequestAminoConverter,
    '/verana.perm.v1.MsgCreateOrUpdatePermissionSession': MsgCreateOrUpdatePermissionSessionAminoConverter,
  });
}

/**
 * Creates a signing client connected to the Verana blockchain using Amino Sign.
 * Matches frontend configuration from veranaChain.sign.client.ts
 * This is the default and matches what the frontend uses.
 */
export async function createSigningClient(
  wallet: Secp256k1HdWallet | DirectSecp256k1HdWallet
): Promise<SigningStargateClient> {
  const registry = createVeranaRegistry();

  // Validate config values before connecting
  if (!config.rpcEndpoint || !config.rpcEndpoint.trim()) {
    throw new Error(`Invalid RPC endpoint: "${config.rpcEndpoint}". Set VERANA_RPC_ENDPOINT environment variable.`);
  }
  if (!config.gasPrice || !config.gasPrice.trim()) {
    throw new Error(`Invalid gas price: "${config.gasPrice}". Set VERANA_GAS_PRICE environment variable.`);
  }

  try {
    const gasPriceObj = GasPrice.fromString(config.gasPrice);

    // Determine if this is an Amino wallet (Secp256k1HdWallet from @cosmjs/amino)
    const isAminoWallet = wallet instanceof Secp256k1HdWallet;

    // Retry connection up to 3 times with exponential backoff
    // This handles cases where the blockchain is still initializing
    const maxRetries = 3;
    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        // Use Amino Sign if wallet is Amino wallet, otherwise use Direct Sign
        const clientOptions: any = {
          registry,
          gasPrice: gasPriceObj,
        };

        if (isAminoWallet) {
          // Add Amino types for Amino Sign (matches frontend)
          clientOptions.aminoTypes = createVeranaAminoTypes();
        }

        const client = await SigningStargateClient.connectWithSigner(
          config.rpcEndpoint,
          wallet,
          clientOptions
        );
        return client;
      } catch (error: any) {
        lastError = error;

        // If it's the "must provide a non-empty value" error, it might be a timing issue
        // Wait before retrying (exponential backoff)
        if (attempt < maxRetries && error.message?.includes("must provide a non-empty value")) {
          const waitTime = Math.pow(2, attempt) * 1000; // 2s, 4s, 8s
          await new Promise(resolve => setTimeout(resolve, waitTime));
          continue;
        }

        // For other errors or last attempt, throw immediately
        throw error;
      }
    }

    // Should never reach here, but just in case
    throw lastError || new Error("Failed to connect after retries");
  } catch (error: any) {
    throw error;
  }
}

/**
 * Creates a query-only client (no signing capability).
 */
export async function createQueryClient(): Promise<StargateClient> {
  return StargateClient.connect(config.rpcEndpoint);
}

/**
 * Helper to get account info from a wallet.
 * Supports both Amino and Direct Sign wallets.
 */
export async function getAccountInfo(wallet: Secp256k1HdWallet | DirectSecp256k1HdWallet) {
  const [account] = await wallet.getAccounts();
  return account;
}

/**
 * Calculate fee using gas simulation (matches frontend approach).
 * The frontend uses client.simulate() to estimate gas, then applies gasAdjustment.
 * This matches the signAndBroadcastManualDirect function in the frontend.
 */
export async function calculateFeeWithSimulation(
  client: SigningStargateClient,
  address: string,
  messages: any[],
  memo: string = ""
) {
  // Simulate gas usage (matches frontend signAndBroadcastManualDirect)
  const simulated = await client.simulate(address, messages, memo);
  const gasLimit = Math.ceil(simulated * config.gasAdjustment);
  const gasPrice = GasPrice.fromString(config.gasPrice);

  // Use calculateFee from @cosmjs/stargate (same as frontend)
  return calculateFee(gasLimit, gasPrice);
}

/**
 * Sign and broadcast with retry logic for unauthorized errors.
 * CRITICAL FIX: Creates a NEW client instance for each transaction (matches frontend).
 *
 * The key difference from previous attempts:
 * 1. We wait for previous transaction to be FULLY confirmed (included in block AND sequence incremented)
 * 2. THEN create a fresh client (fresh cache)
 * 3. THEN getSequence() which will get the correct sequence
 * 4. THEN sign and broadcast
 *
 * This matches the frontend pattern where each transaction gets a fresh client.
 */
export async function signAndBroadcastWithRetry(
  client: SigningStargateClient,
  address: string,
  messages: any[],
  fee: any,
  memo: string = ""
) {
  // Extract wallet from client to create fresh client
  const wallet = (client as any).signer;
  if (!wallet) {
    throw new Error("Cannot extract wallet from client. Client must have a signer.");
  }

  // CRITICAL: Get current on-chain sequence BEFORE creating new client
  // This ensures we know what sequence we should be using
  const queryClientBefore = await createQueryClient();
  let expectedSequence: number;
  try {
    const seqInfo = await queryClientBefore.getSequence(address);
    expectedSequence = seqInfo.sequence;
    console.log(`  🔄 Current on-chain sequence: ${expectedSequence}`);
  } finally {
    queryClientBefore.disconnect();
  }

  // Create fresh client for this transaction (matches frontend signAndBroadcastManualAmino)
  // This gives us a fresh cache, but we still need to ensure sequence is correct
  const freshClient = await createSigningClient(wallet);

  try {
    // Get sequence from fresh client - should match on-chain sequence
    const cachedSeq = await freshClient.getSequence(address);

    // If there's a mismatch, wait a bit and try again
    if (cachedSeq.sequence !== expectedSequence) {
      console.log(`  ⚠️  Sequence mismatch: on-chain=${expectedSequence}, cached=${cachedSeq.sequence}, waiting...`);
      await new Promise((resolve) => setTimeout(resolve, 1000));
      await freshClient.getSequence(address); // Force refresh
    }

    // Match frontend: getSequence() once before signing
    await freshClient.getSequence(address);

    let res = await freshClient.signAndBroadcast(address, messages, fee, memo);

    // If unauthorized error, wait for previous transaction and retry
    const unauthorized = res.code === 4 && typeof res.rawLog === 'string' && res.rawLog.includes('signature verification failed');
    if (unauthorized) {
      console.log(`  ⚠️  Unauthorized error detected. Waiting for previous transaction to confirm...`);

      // CRITICAL: Wait for previous transaction to be FULLY confirmed
      // 1. Wait for it to be included in a block
      // 2. Wait for sequence to increment on-chain
      const queryClientWait = await createQueryClient();
      try {
        // Wait for transaction to be queryable (means it's in a block)
        for (let i = 0; i < 20; i++) {
          await new Promise((resolve) => setTimeout(resolve, 500));
          // Check if sequence has incremented
          const currentSeq = await queryClientWait.getSequence(address);
          if (currentSeq.sequence > expectedSequence) {
            console.log(`  ✓ Previous transaction confirmed, sequence now: ${currentSeq.sequence}`);
            break;
          }
        }
      } finally {
        queryClientWait.disconnect();
      }

      // Wait a bit more to ensure sequence is fully propagated
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // Create another fresh client for retry (fresh cache with updated sequence)
      const retryClient = await createSigningClient(wallet);
      try {
        // Match frontend: getSequence() once before retry
        await retryClient.getSequence(address);
        res = await retryClient.signAndBroadcast(address, messages, fee, memo);
      } finally {
        retryClient.disconnect();
      }
    }

    // Wait for this transaction to be confirmed (helps with next transaction)
    if (res.code === 0) {
      const queryClient = await createQueryClient();
      try {
        // Wait for transaction to be queryable (means it's in a block)
        for (let i = 0; i < 20; i++) {
          try {
            const tx = await queryClient.getTx(res.transactionHash);
            if (tx) {
              break;
            }
          } catch {
            // Transaction not found yet, continue waiting
          }
          await new Promise((resolve) => setTimeout(resolve, 500));
        }
      } finally {
        queryClient.disconnect();
      }
    }

    return res;
  } finally {
    // Always disconnect the fresh client
    freshClient.disconnect();
  }
}

/**
 * Default fee for transactions (fallback if simulation not available).
 * Uses fixed gas limit matching frontend default.
 */
export function getDefaultFee(gas: string = config.gasLimit) {
  // Calculate fee based on gas limit and gas price (matches frontend)
  const gasPriceValue = parseFloat(config.gasPrice.replace("uvna", ""));
  const feeAmount = Math.ceil(parseInt(gas) * gasPriceValue);

  return {
    amount: [{ denom: config.denom, amount: String(feeAmount) }],
    gas,
  };
}

/**
 * Helper to wait for a transaction to be included in a block.
 */
export async function waitForTx(
  client: StargateClient,
  txHash: string,
  timeoutMs: number = 30000
): Promise<void> {
  const startTime = Date.now();
  while (Date.now() - startTime < timeoutMs) {
    try {
      const tx = await client.getTx(txHash);
      if (tx) {
        return;
      }
    } catch {
      // Transaction not found yet, continue waiting
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`Transaction ${txHash} not found within ${timeoutMs}ms`);
}

/**
 * Funds an account from another account using bank send.
 * Uses Direct signing (not Amino) for bank send messages.
 * @param mnemonic - Mnemonic phrase for the funding account
 * @param fromAddress - Address of the account sending funds
 * @param toAddress - Address of the account receiving funds
 * @param amount - Amount to send (e.g., "1000000000uvna")
 * @returns Promise<DeliverTxResponse>
 */
export async function fundAccount(
  mnemonic: string,
  fromAddress: string,
  toAddress: string,
  amount: string
) {
  // Parse amount string like "1000000000uvna" into amount and denom
  // The format is: <number><denom> (e.g., "1000000000uvna")
  let amountValue = "";
  let denom = "";

  for (let i = 0; i < amount.length; i++) {
    if (amount[i] >= '0' && amount[i] <= '9') {
      amountValue += amount[i];
    } else {
      denom = amount.substring(i);
      break;
    }
  }

  if (!amountValue || !denom) {
    throw new Error(`Invalid amount format: ${amount}. Expected format: "1000000000uvna"`);
  }

  // Create a DirectSigningClient for bank send (uses Direct signing, not Amino)
  const { DirectSecp256k1HdWallet } = await import("@cosmjs/proto-signing");
  const { createVeranaRegistry } = await import("./registry");
  const registry = createVeranaRegistry();

  const directWallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: config.addressPrefix,
  });

  // Create DirectSigningClient without Amino types (uses Direct signing)
  const directClient = await SigningStargateClient.connectWithSigner(
    config.rpcEndpoint,
    directWallet,
    {
      registry,
      gasPrice: GasPrice.fromString(config.gasPrice),
      // No aminoTypes - use Direct signing for bank send
    }
  );

  // Use sendTokens which works with Direct signing
  return await directClient.sendTokens(
    fromAddress,
    toAddress,
    [{ denom: denom, amount: amountValue }],
    "auto",
    "Funding account"
  );
}

/**
 * Generates a unique DID for testing.
 */
export function generateUniqueDID(): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8);
  return `did:verana:test:${timestamp}:${random}`;
}

/**
 * Gets the current block time from the blockchain.
 * This is important because the blockchain uses block time, not local time.
 */
export async function getBlockTime(client: StargateClient): Promise<Date> {
  const block = await client.getBlock();
  // block.header.time might be a string or Date, convert to Date
  const time = block.header.time as any;
  if (time instanceof Date) {
    return time;
  }
  return new Date(time);
}

/**
 * Waits until the blockchain's block time has passed the given time.
 * This ensures permissions are effective according to blockchain time, not local time.
 */
export async function waitUntilBlockTime(
  client: StargateClient,
  targetTime: Date,
  maxWaitMs: number = 30000
): Promise<void> {
  const startTime = Date.now();
  while (Date.now() - startTime < maxWaitMs) {
    const blockTime = await getBlockTime(client);
    if (blockTime >= targetTime) {
      return;
    }
    // Wait a bit before checking again
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`Block time did not reach ${targetTime.toISOString()} within ${maxWaitMs}ms`);
}

/**
 * Waits for a permission to become effective by checking blockchain block time.
 * This is needed because permissions are created with effective_from in the future,
 * and operations like Extend/Revoke require the permission to be effective.
 */
export async function waitForPermissionToBecomeEffective(
  client: StargateClient,
  effectiveFrom: Date,
  maxWaitMs: number = 30000
): Promise<void> {
  const startTime = Date.now();
  let lastBlockTime: Date | null = null;

  while (Date.now() - startTime < maxWaitMs) {
    const blockTime = await getBlockTime(client);
    lastBlockTime = blockTime;

    // Check if block time has passed effective_from
    if (blockTime >= effectiveFrom) {
      return;
    }

    // Wait a bit before checking again (check every second)
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  // If we timeout, provide helpful error message
  const timeRemaining = effectiveFrom.getTime() - (lastBlockTime?.getTime() || Date.now());
  throw new Error(
    `Permission not yet effective. Block time: ${lastBlockTime?.toISOString()}, ` +
    `effective_from: ${effectiveFrom.toISOString()}, ` +
    `time remaining: ${Math.ceil(timeRemaining / 1000)}s`
  );
}

/**
 * Waits for the account sequence to propagate after a transaction.
 * Uses polling with exponential backoff instead of hardcoded waits.
 * This handles race conditions where the sequence may not have incremented yet.
 *
 * @param client - The signing client to use for sequence queries
 * @param address - The account address to check sequence for
 * @param expectedSequence - Optional expected sequence number. If provided, waits until
 *                          the sequence is greater than this value. If not provided,
 *                          just refreshes the sequence cache multiple times.
 * @param maxWaitMs - Maximum time to wait in milliseconds (default: 60000 = 60s)
 * @returns Promise<number> - The current sequence number after propagation
 */
export async function waitForSequencePropagation(
  client: SigningStargateClient,
  address: string,
  expectedSequence?: number,
  maxWaitMs: number = 60000
): Promise<number> {
  const startTime = Date.now();
  let pollInterval = 500; // Start with 500ms
  const maxPollInterval = 2000; // Cap at 2s
  let lastSequence = 0;

  console.log(`  ⏳ Polling for sequence propagation (timeout: ${maxWaitMs / 1000}s)...`);

  while (Date.now() - startTime < maxWaitMs) {
    try {
      // Force refresh sequence from the chain
      const seqInfo = await client.getSequence(address);
      lastSequence = seqInfo.sequence;

      // If no expected sequence, do multiple refreshes to ensure cache is fully updated
      // This handles rapid multi-transaction scenarios where cache propagation may lag
      if (expectedSequence === undefined) {
        // Do 3 refreshes with 1-second gaps to ensure full propagation
        for (let refreshCount = 0; refreshCount < 3; refreshCount++) {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          const refreshedSeq = await client.getSequence(address);
          lastSequence = refreshedSeq.sequence;
        }
        console.log(`  ✓ Sequence cache refreshed (3x), current sequence: ${lastSequence}`);
        return lastSequence;
      }

      // If we have an expected sequence, wait until current > expected
      if (lastSequence > expectedSequence) {
        console.log(`  ✓ Sequence propagated: ${expectedSequence} -> ${lastSequence}`);
        return lastSequence;
      }

      // Wait before polling again (exponential backoff)
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
      pollInterval = Math.min(pollInterval * 1.5, maxPollInterval);

    } catch (error: any) {
      // If there's an error querying sequence, wait and retry
      console.log(`  ⚠️  Error querying sequence: ${error.message}, retrying...`);
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }
  }

  // Timeout reached
  if (expectedSequence !== undefined) {
    throw new Error(
      `Sequence propagation timeout after ${maxWaitMs / 1000}s. ` +
      `Expected sequence > ${expectedSequence}, but current is ${lastSequence}`
    );
  }

  console.log(`  ⚠️  Sequence propagation timeout, but continuing with sequence: ${lastSequence}`);
  return lastSequence;
}
