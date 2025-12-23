/**
 * Verana Client Helper
 * Provides utilities for connecting to the Verana blockchain.
 */

import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { SigningStargateClient, StargateClient, GasPrice, calculateFee } from "@cosmjs/stargate";
import { createVeranaRegistry } from "./registry";

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
 * Creates a wallet from a mnemonic phrase.
 */
export async function createWallet(mnemonic: string): Promise<DirectSecp256k1HdWallet> {
  return DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: config.addressPrefix,
  });
}

/**
 * Creates a signing client connected to the Verana blockchain.
 * Matches frontend configuration from veranaChain.sign.client.ts
 */
export async function createSigningClient(
  wallet: DirectSecp256k1HdWallet
): Promise<SigningStargateClient> {
  const registry = createVeranaRegistry();

  // Validate config values before connecting
  if (!config.rpcEndpoint || !config.rpcEndpoint.trim()) {
    throw new Error(`Invalid RPC endpoint: "${config.rpcEndpoint}". Set VERANA_RPC_ENDPOINT environment variable.`);
  }
  if (!config.gasPrice || !config.gasPrice.trim()) {
    throw new Error(`Invalid gas price: "${config.gasPrice}". Set VERANA_GAS_PRICE environment variable.`);
  }

  return SigningStargateClient.connectWithSigner(config.rpcEndpoint, wallet, {
    registry,
    gasPrice: GasPrice.fromString(config.gasPrice),
  });
}

/**
 * Creates a query-only client (no signing capability).
 */
export async function createQueryClient(): Promise<StargateClient> {
  return StargateClient.connect(config.rpcEndpoint);
}

/**
 * Helper to get account info from a wallet.
 */
export async function getAccountInfo(wallet: DirectSecp256k1HdWallet) {
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
