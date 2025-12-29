/**
 * Verana Client Helper
 * Provides utilities for connecting to the Verana blockchain.
 */

import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Secp256k1HdWallet } from "@cosmjs/amino";
import { SigningStargateClient, StargateClient, GasPrice, calculateFee, AminoTypes } from "@cosmjs/stargate";
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
 * Sign and broadcast with retry logic for unauthorized errors (matches frontend).
 * The frontend's signAndBroadcastManualAmino retries once if it gets an unauthorized error,
 * as this can happen due to account number/sequence timing issues.
 */
export async function signAndBroadcastWithRetry(
  client: SigningStargateClient,
  address: string,
  messages: any[],
  fee: any,
  memo: string = ""
) {
  // Fetch account sequence & accountNumber (like frontend does)
  // This ensures the client has the latest sequence cached
  // Add a small delay before fetching to ensure any previous transaction is processed
  await new Promise((resolve) => setTimeout(resolve, 300));
  const sequenceBefore = await client.getSequence(address);
  
  let res = await client.signAndBroadcast(address, messages, fee, memo);
  
  // If unauthorized error, retry once (matches frontend signAndBroadcastManualAmino)
  const unauthorized = res.code === 4 && typeof res.rawLog === 'string' && res.rawLog.includes('signature verification failed');
  if (unauthorized) {
    
    // Add a longer delay to ensure previous transaction is fully processed and sequence is updated
    await new Promise((resolve) => setTimeout(resolve, 2000));
    // Refresh account sequence before retry - this forces a fresh fetch
    // Call it twice to ensure cache is cleared
    const seq1 = await client.getSequence(address);
    await new Promise((resolve) => setTimeout(resolve, 500));
    const seq2 = await client.getSequence(address);
    res = await client.signAndBroadcast(address, messages, fee, memo);
  }
  
  return res;
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
