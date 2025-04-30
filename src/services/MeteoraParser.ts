import type {
  ParsedTransactionWithMeta,
  PublicKey,
  ParsedInstruction,
  PartiallyDecodedInstruction,
} from "@solana/web3.js";
import type { TransactionData } from "../types";
import { BorshInstructionCoder, type Idl } from "@coral-xyz/anchor";
import { IDL } from "@meteora-ag/dlmm";
import bs58 from "bs58";

// Constants for Meteora program IDs (v1 and v2)
export const METEORA_PROGRAM_ID = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo";
export const METEORA_PROGRAM_ID_V2 =
  "dLMWnWVBcCD98KYSgCvTftbHZzbndhK7GhMtjJ7jAqjF";
export const METEORA_PROGRAM_IDS = [
  METEORA_PROGRAM_ID, // v1 mainnet
  METEORA_PROGRAM_ID_V2, // v2 mainnet
  "GFXsSL5xWRGWEFx3KjqR3VBSagwrwDcnwgoBqwFQxVac", // Alternative/devnet
];

// Known token programs
export const TOKEN_PROGRAM_ID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA";
export const ASSOCIATED_TOKEN_PROGRAM_ID =
  "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL";

// SOL token address
export const SOL_TOKEN_MINT = "So11111111111111111111111111111111111111112";

// Some known Meteora-related tokens (to help identify Meteora transactions)
export const METEORA_RELATED_TOKENS = [
  "LBooNyXDsHPxfJzwdj8hGJy8ww32cLqiKAhh75pL5JF", // Meteora Token
];

// Some known Meteora-related accounts (to help identify Meteora transactions)
export const METEORA_RELATED_ACCOUNTS = [
  "Eo5YeDSgnj4RZ3LFE46YHoCWxVDn2qhqQf4NTkVECN65", // Meteora fee recipient
];

// Meteora LP position account minimum size (rough estimate)
// This helps identify position accounts
const MIN_POSITION_ACCOUNT_SIZE = 300;

// Meteora LB pair account minimum size
const MIN_PAIR_ACCOUNT_SIZE = 500;

// Instruction types
export type MeteoraDlmmInstructionType =
  | "open"
  | "add"
  | "remove"
  | "claim"
  | "close"
  | "swap";

// Instruction names
export type MeteoraDlmmInstructionName =
  | "initializePosition"
  | "addLiquidity"
  | "addLiquidityByWeight"
  | "addLiquidityByStrategy"
  | "addLiquidityByStrategyOneSide"
  | "addLiquidityOneSide"
  | "removeLiquidity"
  | "removeAllLiquidity"
  | "removeLiquiditySingleSide"
  | "removeLiquidityByRange"
  | "claimFee"
  | "closePosition"
  | "unknownMeteora"
  | "swap"
  | "swapV2";

// Map of instruction names to instruction types
const INSTRUCTION_TYPE_MAP: Record<string, MeteoraDlmmInstructionType> = {
  // Position management
  initializePosition: "open",
  initializePositionByWeight: "open",
  initializePositionByStrategy: "open",

  // Add liquidity
  addLiquidity: "add",
  addLiquidityByWeight: "add",
  addLiquidityByStrategy: "add",
  addLiquidityByStrategyOneSide: "add",
  addLiquidityOneSide: "add",

  // Remove liquidity
  removeLiquidity: "remove",
  removeAllLiquidity: "remove",
  removeLiquiditySingleSide: "remove",
  removeLiquidityByRange: "remove",

  // Fee claiming
  claimFee: "claim",
  claimSwapFee: "claim",

  // Close position
  closePosition: "close",

  // Swaps
  swap: "swap",
  swapV2: "swap",

  // Default fallback
  unknownMeteora: "add",
};

// Account structure for Meteora operations
interface MeteoraDlmmAccounts {
  position: string;
  lbPair: string;
  sender: string;
}

// Token transfer information
export interface TokenTransferInfo {
  mint: string;
  amount: number;
  direction: "in" | "out";
}

// Interface for Meteora instructions
export interface MeteoraDlmmInstruction {
  signature: string;
  slot: number;
  blockTime: number;
  programId: string; // Added to track which version of Meteora
  instructionName: string;
  instructionType: MeteoraDlmmInstructionType;
  accounts: MeteoraDlmmAccounts;
  tokenTransfers: TokenTransferInfo[];
  activeBinId?: number;
  removalBps?: number;
  decodedData?: Record<string, unknown>; // Store decoded instruction data when available
}

// Define a more specific RPC Response interface
interface RpcResponse {
  jsonrpc: string;
  id: number;
  result?: ParsedTransactionWithMeta;
  error?: {
    code: number;
    message: string;
  };
}

// Initialize instruction coder (v1 - works for both v1 and v2 as they share most instructions)
// We need to use 'any' here because the IDL from the SDK doesn't match the Anchor Idl type exactly
const instructionCoderV1 = new BorshInstructionCoder(IDL as unknown as Idl);

/**
 * Sort Meteora instructions by time and type
 */
export function sortMeteoraInstructions(
  instructions: MeteoraDlmmInstruction[]
): MeteoraDlmmInstruction[] {
  return instructions.sort((a, b) => {
    return a.blockTime !== b.blockTime
      ? // If they're different block times, take them in ascending order
        a.blockTime - b.blockTime
      : // Take the first instruction when it is open
        a.instructionType === "open"
        ? -1
        : // Take the second instruction when it is open
          b.instructionType === "open"
          ? 1
          : // Take the first instruction when it is claim
            a.instructionType === "claim"
            ? -1
            : // Take the second instruction when it is claim
              b.instructionType === "claim"
              ? 1
              : // Take the second instruction when the first is close
                a.instructionType === "close"
                ? 1
                : // Take the first instruction when the second is close
                  b.instructionType === "close"
                  ? -1
                  : // Everything else just take it as it comes
                    0;
  });
}

/**
 * Attempt to decode a Meteora instruction with Anchor's BorshInstructionCoder
 */
function decodeInstruction(
  instructionData: Buffer | string,
  programId: string
): { name: string; data: Record<string, unknown> } | null {
  try {
    if (!instructionData) return null;

    // Attempt to decode using Anchor's BorshInstructionCoder
    const decoded = instructionCoderV1.decode(
      typeof instructionData === "string"
        ? Buffer.from(bs58.decode(instructionData))
        : instructionData
    );

    if (decoded) {
      return {
        name: decoded.name,
        data: decoded.data as Record<string, unknown>,
      };
    }

    return null;
  } catch (error) {
    console.log(`Failed to decode instruction: ${error}`);
    return null;
  }
}

/**
 * Extract token transfers from a transaction
 */
function extractTokenTransfers(
  transaction: ParsedTransactionWithMeta
): TokenTransferInfo[] {
  try {
    const transfers: TokenTransferInfo[] = [];

    // Check pre/post token balances to infer transfers
    const preBalances = transaction.meta?.preTokenBalances || [];
    const postBalances = transaction.meta?.postTokenBalances || [];

    if (preBalances.length === 0 && postBalances.length === 0) {
      return [];
    }

    // Process all token accounts that exist in both pre and post balances
    for (const post of postBalances) {
      if (!post?.mint) continue;

      const pre = preBalances.find((p) => p.accountIndex === post.accountIndex);

      // If account exists in both pre and post
      if (pre) {
        const preAmount = Number(pre.uiTokenAmount.amount) || 0;
        const postAmount = Number(post.uiTokenAmount.amount) || 0;
        const diff = postAmount - preAmount;

        if (Math.abs(diff) > 0.000001) {
          // Use a small epsilon to handle floating point errors
          transfers.push({
            mint: post.mint,
            amount: Math.abs(diff),
            direction: diff > 0 ? "in" : "out",
          });
        }
      } else {
        // New token account - consider it as receiving tokens
        const amount = Number(post.uiTokenAmount.amount) || 0;
        if (amount > 0) {
          transfers.push({
            mint: post.mint,
            amount,
            direction: "in",
          });
        }
      }
    }

    // Check for token accounts that were closed (existed in pre but not in post)
    for (const pre of preBalances) {
      if (!pre?.mint) continue;

      const post = postBalances.find(
        (p) => p.accountIndex === pre.accountIndex
      );

      // If account existed in pre but not in post, it was closed/emptied
      if (!post) {
        const amount = Number(pre.uiTokenAmount.amount) || 0;
        if (amount > 0) {
          transfers.push({
            mint: pre.mint,
            amount,
            direction: "out",
          });
        }
      }
    }

    return transfers;
  } catch (error) {
    console.error("Error extracting token transfers:", error);
    return [];
  }
}

// Define a more specific type for account keys
type AccountKey = { pubkey?: { toString(): string } } | { toString(): string };

/**
 * Extract account information for a Meteora instruction
 */
function extractAccountInfo(
  accountKeys: AccountKey[],
  instruction: {
    parsed?: {
      info?: { source?: string; destination?: string; authority?: string };
    };
    accounts?: Array<number>;
    pubkeys?: Array<PublicKey>;
  },
  programId: string
): MeteoraDlmmAccounts {
  try {
    const accounts: MeteoraDlmmAccounts = {
      position: "unknown",
      lbPair: "unknown",
      sender: "unknown",
    };

    // For parsed instructions
    if (instruction.parsed?.info) {
      const info = instruction.parsed.info;

      // Source/destination for token transfers can hint at position/sender
      if (info.source) accounts.sender = info.source;
      if (info.destination) accounts.position = info.destination;

      // Authority is often the sender
      if (info.authority) accounts.sender = info.authority;
    }

    // For raw instructions with account indexes
    if (instruction.accounts) {
      // Try to get sender (usually first account)
      if (instruction.accounts.length > 0) {
        const senderIndex = instruction.accounts[0];
        const sender =
          senderIndex !== undefined ? accountKeys[senderIndex] : undefined;
        if (sender) {
          if ("pubkey" in sender && sender.pubkey) {
            accounts.sender = sender.pubkey.toString();
          } else {
            accounts.sender = sender.toString();
          }
        }
      }

      // Position is typically the 2nd account for position operations
      if (instruction.accounts.length > 1) {
        const posIndex = instruction.accounts[1];
        const position =
          posIndex !== undefined ? accountKeys[posIndex] : undefined;
        if (position) {
          if ("pubkey" in position && position.pubkey) {
            accounts.position = position.pubkey.toString();
          } else {
            accounts.position = position.toString();
          }
        }
      }

      // LB Pair is typically the 3rd account
      if (instruction.accounts.length > 2) {
        const lbPairIndex = instruction.accounts[2];
        const lbPair =
          lbPairIndex !== undefined ? accountKeys[lbPairIndex] : undefined;
        if (lbPair) {
          if ("pubkey" in lbPair && lbPair.pubkey) {
            accounts.lbPair = lbPair.pubkey.toString();
          } else {
            accounts.lbPair = lbPair.toString();
          }
        }
      }
    }
    // For raw instructions with PublicKey arrays (PartiallyDecodedInstruction)
    else if (instruction.pubkeys) {
      // Try to get sender (usually first account)
      if (instruction.pubkeys.length > 0 && instruction.pubkeys[0]) {
        accounts.sender = instruction.pubkeys[0].toString();
      }

      // Position is typically the 2nd account for position operations
      if (instruction.pubkeys.length > 1 && instruction.pubkeys[1]) {
        accounts.position = instruction.pubkeys[1].toString();
      }

      // LB Pair is typically the 3rd account
      if (instruction.pubkeys.length > 2 && instruction.pubkeys[2]) {
        accounts.lbPair = instruction.pubkeys[2].toString();
      }
    }

    return accounts;
  } catch (error) {
    console.error("Error extracting account info:", error);
    return {
      position: "error",
      lbPair: "error",
      sender: "error",
    };
  }
}

/**
 * Extract Meteora instructions from transaction data
 */
export function extractMeteoraInstructions(
  transaction: ParsedTransactionWithMeta
): MeteoraDlmmInstruction[] {
  try {
    const txSignature =
      transaction.transaction.signatures[0]?.slice(0, 8) || "unknown";

    if (!transaction.blockTime) {
      return [];
    }

    // Track all Meteora instructions found
    const meteoraInstructions: MeteoraDlmmInstruction[] = [];
    // Account keys from the transaction
    const accountKeys = transaction.transaction.message.accountKeys;

    // Define a generic instruction type that works for both parsed and partially decoded
    type GenericInstruction = {
      programId?: { toString(): string };
      program?: string;
      data?: Buffer | string;
      accounts?: Array<number>;
      pubkeys?: Array<PublicKey>;
      parsed?: { type?: string; info?: Record<string, unknown> };
    };

    // Entry in our instructions array
    type InstructionEntry = {
      instruction: GenericInstruction;
      type: "main" | "inner";
      innerIndex?: number;
    };

    const allInstructions: InstructionEntry[] = [];

    // Add main instructions
    for (const ix of transaction.transaction.message.instructions) {
      // Convert to our generic instruction interface
      allInstructions.push({
        instruction: ix as unknown as GenericInstruction,
        type: "main",
      });
    }

    // Add inner instructions
    if (transaction.meta?.innerInstructions) {
      for (const [
        groupIndex,
        innerGroup,
      ] of transaction.meta.innerInstructions.entries()) {
        for (const ix of innerGroup.instructions) {
          // Convert to our generic instruction interface
          allInstructions.push({
            instruction: ix as unknown as GenericInstruction,
            type: "inner",
            innerIndex: groupIndex,
          });
        }
      }
    }

    // Look for Meteora program IDs in all instructions
    let hasMeteora = false;
    const foundInstructions: string[] = [];

    for (const { instruction, type, innerIndex } of allInstructions) {
      // Get program ID
      let programId: string | null = null;
      if (instruction.programId) {
        programId = instruction.programId.toString();
      } else if (instruction.program === "spl-token") {
        programId = TOKEN_PROGRAM_ID;
      }

      // Check if it's a Meteora program
      if (programId && METEORA_PROGRAM_IDS.includes(programId)) {
        hasMeteora = true;

        // Try to decode the instruction
        let instructionName = "unknownMeteora";
        let decodedData: Record<string, unknown> | null = null;

        // If we have data, attempt to decode it
        if (instruction.data) {
          const decoded = decodeInstruction(instruction.data, programId);
          if (decoded) {
            instructionName = decoded.name;
            decodedData = decoded.data;
            foundInstructions.push(instructionName);
          }
        }

        // Get instruction type from the map
        const instructionType = INSTRUCTION_TYPE_MAP[instructionName] || "add";

        // Extract account information
        const accounts = extractAccountInfo(
          accountKeys,
          instruction,
          programId
        );

        // Extract token transfers for this transaction
        const tokenTransfers = extractTokenTransfers(transaction);

        // Create the instruction object with optional fields properly typed
        const meteoraInstruction: MeteoraDlmmInstruction = {
          signature: transaction.transaction.signatures[0] || "",
          slot: transaction.slot,
          blockTime: transaction.blockTime,
          programId,
          instructionName,
          instructionType,
          accounts,
          tokenTransfers,
          // Only include optional fields if they have values
          ...(decodedData ? { decodedData } : {}),
        };

        meteoraInstructions.push(meteoraInstruction);
      }
    }

    // If no Meteora program found, try to infer from token transfers and accounts
    if (!hasMeteora) {
      // Extract token transfers
      const tokenTransfers = extractTokenTransfers(transaction);

      // Save all account keys for analysis
      const allAccountKeys = transaction.transaction.message.accountKeys.map(
        (key) => (key.pubkey ? key.pubkey.toString() : key.toString())
      );

      // Check if any known Meteora accounts are involved
      const hasMeteoraDependentAccounts = allAccountKeys.some((key) =>
        METEORA_RELATED_ACCOUNTS.includes(key)
      );

      // Check if any known Meteora tokens are involved
      const hasMeteoraDependentTokens = tokenTransfers.some((t) =>
        METEORA_RELATED_TOKENS.includes(t.mint)
      );

      // If we have SOL token or specific patterns of transfers, it could be a Meteora-related transaction
      const hasSolTransfers = tokenTransfers.some(
        (t) =>
          t.mint === SOL_TOKEN_MINT ||
          t.mint.startsWith("So111111111111111111111111111111111111111")
      );

      // Check for pairs of token transfers (common in swaps)
      const uniqueMints = new Set(tokenTransfers.map((t) => t.mint)).size;
      const hasMultipleTokens = uniqueMints > 1;

      if (
        (hasMeteoraDependentAccounts ||
          hasMeteoraDependentTokens ||
          hasSolTransfers ||
          hasMultipleTokens) &&
        tokenTransfers.length > 0
      ) {
        // Determine if this looks like a swap, add, or remove liquidity
        let instructionType: MeteoraDlmmInstructionType = "add";
        let instructionName = "unknownMeteora";

        // Infer the instruction type from token transfer patterns
        const inTokens = tokenTransfers.filter((t) => t.direction === "in");
        const outTokens = tokenTransfers.filter((t) => t.direction === "out");

        if (inTokens.length === 1 && outTokens.length === 1) {
          // One token in, one token out - likely a swap
          instructionType = "swap";
          instructionName = "swap";
        } else if (outTokens.length > 0 && inTokens.length === 0) {
          // Only tokens going out - adding liquidity
          instructionType = "add";
          instructionName = "addLiquidity";
        } else if (inTokens.length > 0 && outTokens.length === 0) {
          // Only tokens coming in - removing liquidity or claiming fees
          instructionType = "remove";
          instructionName = "removeLiquidity";
        }

        // Default to v2 if we detect it might be a Meteora transaction but don't know which version
        const detectedProgramId =
          hasMeteoraDependentAccounts || hasMeteoraDependentTokens
            ? METEORA_PROGRAM_ID_V2 // Prefer V2 for newer transactions
            : METEORA_PROGRAM_ID; // Default to V1 otherwise

        foundInstructions.push(`Inferred ${instructionName}`);

        // Create a Meteora instruction with the inferred type
        const meteoraInstruction: MeteoraDlmmInstruction = {
          signature: transaction.transaction.signatures[0] || "",
          slot: transaction.slot,
          blockTime: transaction.blockTime,
          programId: detectedProgramId,
          instructionName,
          instructionType,
          accounts: {
            position: "inferred",
            lbPair: "inferred",
            sender: "inferred",
          },
          tokenTransfers,
        };

        meteoraInstructions.push(meteoraInstruction);
      }
    }

    // Only log if we found Meteora instructions
    if (meteoraInstructions.length > 0) {
      // Log each unique instruction found
      if (foundInstructions.length > 0) {
        for (const instruction of foundInstructions) {
          console.log(`${txSignature}: ${instruction}`);
        }
      }

      console.log(
        `${txSignature}: Found ${meteoraInstructions.length} Meteora instructions`
      );
    }

    return meteoraInstructions;
  } catch (error) {
    console.error("Error parsing Meteora instructions:", error);
    return [];
  }
}

/**
 * Process transactions to find Meteora operations
 */
export function processMeteoraTransactions(
  transactions: TransactionData[],
  onProgress?: (status: string, current: number, total: number) => void
): Promise<MeteoraDlmmInstruction[]> {
  return new Promise((resolve) => {
    const meteoraInstructions: MeteoraDlmmInstruction[] = [];
    let processed = 0;
    let meteoraTransactions = 0;

    console.log(
      `Processing ${transactions.length} transactions for Meteora operations`
    );

    // Process transactions in batches of 20
    const batchSize = 20;
    const batches = Math.ceil(transactions.length / batchSize);

    const processNextBatch = async (batchIndex: number) => {
      if (batchIndex >= batches) {
        console.log(
          `Found ${meteoraInstructions.length} Meteora instructions in ${meteoraTransactions} transactions`
        );

        // Count by instruction type
        const counts: Record<MeteoraDlmmInstructionType, number> = {
          open: 0,
          add: 0,
          remove: 0,
          claim: 0,
          close: 0,
          swap: 0,
        };

        for (const ix of meteoraInstructions) {
          counts[ix.instructionType] = (counts[ix.instructionType] || 0) + 1;
        }

        // Only log the counts if we found instructions
        if (meteoraInstructions.length > 0) {
          console.log("Instruction types:");
          for (const [type, count] of Object.entries(counts)) {
            if (count > 0) {
              console.log(` - ${type}: ${count}`);
            }
          }
        }

        if (onProgress) {
          onProgress(
            "Processed Meteora transactions",
            transactions.length,
            transactions.length
          );
        }
        resolve(sortMeteoraInstructions(meteoraInstructions));
        return;
      }

      const start = batchIndex * batchSize;
      const end = Math.min(start + batchSize, transactions.length);
      const batch = transactions.slice(start, end);

      if ((batchIndex + 1) % 5 === 0) {
        console.log(
          `Progress: ${Math.round(((batchIndex + 1) / batches) * 100)}%`
        );
      }

      // Process each transaction in the batch
      for (const tx of batch) {
        try {
          const response = await fetch("https://api.mainnet-beta.solana.com", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              jsonrpc: "2.0",
              id: 1,
              method: "getTransaction",
              params: [
                tx.signature,
                { encoding: "json", maxSupportedTransactionVersion: 0 },
              ],
            }),
          });

          const jsonResponse = (await response.json()) as RpcResponse;

          if (jsonResponse.result) {
            const instructions = extractMeteoraInstructions(
              jsonResponse.result
            );

            if (instructions.length > 0) {
              meteoraTransactions++;
              meteoraInstructions.push(...instructions);
            }
          } else if (
            jsonResponse.error?.code &&
            jsonResponse.error.code !== -32602
          ) {
            // Only log serious errors (not just transaction not found)
            console.log(
              `Error (${tx.signature.slice(0, 8)}): ${jsonResponse.error.message}`
            );
          }
        } catch (error) {
          console.error(
            `Error processing transaction ${tx.signature.slice(0, 8)}:`,
            error
          );
        }

        processed++;
        if (processed % 20 === 0 && onProgress) {
          onProgress(
            "Processing Meteora transactions",
            processed,
            transactions.length
          );
        }
      }

      // Process next batch
      await processNextBatch(batchIndex + 1);
    };

    // Start processing batches
    processNextBatch(0);
  });
}

// For a real implementation, a proper Anchor-based parser would be used:
/*
import { IDL, LBCLMM_PROGRAM_IDS } from "@meteora-ag/dlmm";
import {
  type Idl,
  type Instruction,
  BorshEventCoder,
  BorshInstructionCoder,
} from "@project-serum/anchor";
import { base64, bs58 } from "@project-serum/anchor/dist/cjs/utils/bytes";

export class MeteoraTransactionParser {
  private instructionCoder: BorshInstructionCoder;
  private eventCoder: BorshEventCoder;
  
  constructor() {
    this.instructionCoder = new BorshInstructionCoder(IDL as Idl);
    this.eventCoder = new BorshEventCoder(IDL as Idl);
  }
  
  public parseTransaction(transaction: ParsedTransactionWithMeta): MeteoraDlmmInstruction[] {
    // Implementation will go here when we have the required dependencies
    return [];
  }
  
  private decodeInstruction(
    instruction: PartiallyDecodedInstruction
  ): MeteoraDlmmDecodedInstruction | null {
    // Implementation will go here
    return null;
  }
  
  private getTokenTransfers(
    transaction: ParsedTransactionWithMeta,
    instructionIndex: number
  ): TokenTransferInfo[] {
    // Implementation will go here
    return [];
  }
  
  private getPositionAccounts(
    decodedInstruction: MeteoraDlmmDecodedInstruction,
    accountMetas: AccountMeta[]
  ): MeteoraDlmmAccounts {
    // Implementation will go here
    return { position: "", lbPair: "", sender: "" };
  }
  
  private getActiveBinId(
    transaction: ParsedTransactionWithMeta,
    index: number
  ): number | null {
    // Implementation will go here
    return null;
  }
  
  private getRemovalBps(
    decodedInstruction: MeteoraDlmmDecodedInstruction
  ): number | null {
    // Implementation will go here
    return null;
  }
}
*/
