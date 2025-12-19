/**
 * Verana Client Helper
 * Provides utilities for connecting to the Verana blockchain.
 */

import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { SigningStargateClient, StargateClient, GasPrice, calculateFee } from "@cosmjs/stargate";
import { createVeranaRegistry } from "./registry";

// Default configuration - can be overridden via environment variables
// Matches frontend configuration from veranaChain.sign.client.ts
export const config = {
  rpcEndpoint: process.env.VERANA_RPC_ENDPOINT || "http://localhost:26657",
  lcdEndpoint: process.env.VERANA_LCD_ENDPOINT || "http://localhost:1317",
  chainId: process.env.VERANA_CHAIN_ID || "verana",
  addressPrefix: process.env.VERANA_ADDRESS_PREFIX || "verana",
  denom: process.env.VERANA_DENOM || "uvna",
  gasPrice: process.env.VERANA_GAS_PRICE || "3uvna", // Matches frontend
  gasLimit: process.env.VERANA_GAS_LIMIT || "300000", // Matches frontend
  gasAdjustment: parseFloat(process.env.VERANA_GAS_ADJUSTMENT || "2"), // Matches frontend
};

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
