import type { ParsedTransactionWithMeta, PublicKey } from "@solana/web3.js";
import type { TransactionData } from "../types";

// Constants for Meteora program ID
export const METEORA_PROGRAM_ID = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo";

// Known token programs
export const TOKEN_PROGRAM_ID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA";
export const ASSOCIATED_TOKEN_PROGRAM_ID =
  "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL";

// SOL token address
export const SOL_TOKEN_MINT = "So11111111111111111111111111111111111111112";

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
  | "close";

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
  | "unknownMeteora";

// Map of instruction names to instruction types
const INSTRUCTION_MAP: Record<string, MeteoraDlmmInstructionType> = {
  initializePosition: "open",
  addLiquidity: "add",
  addLiquidityByWeight: "add",
  addLiquidityByStrategy: "add",
  addLiquidityByStrategyOneSide: "add",
  addLiquidityOneSide: "add",
  removeLiquidity: "remove",
  removeAllLiquidity: "remove",
  removeLiquiditySingleSide: "remove",
  removeLiquidityByRange: "remove",
  claimFee: "claim",
  closePosition: "close",
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
  instructionName: string;
  instructionType: MeteoraDlmmInstructionType;
  accounts: MeteoraDlmmAccounts;
  tokenTransfers: TokenTransferInfo[];
  activeBinId: number | null;
  removalBps: number | null;
}

// Define an interface for instructions
interface InstructionWithProgramId {
  programId: { toString(): string };
  data?: string | Uint8Array;
  accounts?: number[];
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
 * Attempt to identify the Meteora instruction type based on token transfers
 */
function identifyInstructionType(tokenTransfers: TokenTransferInfo[]): {
  instructionName: MeteoraDlmmInstructionName;
  instructionType: MeteoraDlmmInstructionType;
} {
  // No transfers - likely an initialization
  if (tokenTransfers.length === 0) {
    return {
      instructionName: "initializePosition",
      instructionType: "open",
    };
  }

  // Get counts of transfers by direction
  const inTransfers = tokenTransfers.filter((t) => t.direction === "in");
  const outTransfers = tokenTransfers.filter((t) => t.direction === "out");
  const inCount = inTransfers.length;
  const outCount = outTransfers.length;

  // Get unique token mints involved
  const uniqueMints = new Set(tokenTransfers.map((t) => t.mint));

  // Check for SOL token (So111111...) received or sent
  const solTransfers = tokenTransfers.filter(
    (t) =>
      t.mint === SOL_TOKEN_MINT ||
      t.mint.startsWith("So111111111111111111111111111111111111111")
  );
  const solReceived = solTransfers.some((t) => t.direction === "in");
  const solSent = solTransfers.some((t) => t.direction === "out");

  // PATTERN: Remove liquidity - typically receive SOL without sending anything
  if (solReceived && inCount > 0 && outCount === 0) {
    return {
      instructionName: "removeLiquidity",
      instructionType: "remove",
    };
  }

  // PATTERN: Detect token swap operations - typically same token both in and out
  const sameMintInAndOut = [...uniqueMints].some(
    (mint) =>
      tokenTransfers.some((t) => t.mint === mint && t.direction === "in") &&
      tokenTransfers.some((t) => t.mint === mint && t.direction === "out")
  );

  if (sameMintInAndOut) {
    // If we have a token going both in and out, and SOL going out,
    // it's likely a swap or position operation
    if (solSent) {
      return {
        instructionName: "addLiquidityByStrategy",
        instructionType: "add",
      };
    }

    // If no SOL involved, could be a claim
    return {
      instructionName: "claimFee",
      instructionType: "claim",
    };
  }

  // PATTERN: Add liquidity - typically send SOL and receive tokens
  if (solSent && outCount > 0) {
    return {
      instructionName: "addLiquidity",
      instructionType: "add",
    };
  }

  // PATTERN: Claim fees - typically receive more tokens than you send
  if (inCount > outCount && solReceived) {
    return {
      instructionName: "claimFee",
      instructionType: "claim",
    };
  }

  // PATTERN: Close position - typically have same token in both directions
  // or same number of tokens in/out but with different tokens
  if (inCount === outCount) {
    // Check if we're transferring the same tokens in both directions
    const inMints = new Set(inTransfers.map((t) => t.mint));
    const outMints = new Set(outTransfers.map((t) => t.mint));

    // If we have the same tokens going in and out
    const commonMints = [...inMints].filter((mint) => outMints.has(mint));

    if (commonMints.length > 0 || uniqueMints.size < tokenTransfers.length) {
      return {
        instructionName: "closePosition",
        instructionType: "close",
      };
    }
  }

  // PATTERN: Add liquidity - only sending tokens (outbound transfers only)
  if (outCount > 0 && inCount === 0) {
    return {
      instructionName: "addLiquidity",
      instructionType: "add",
    };
  }

  // PATTERN: Add liquidity with one-sided strategy
  if (uniqueMints.size === 1 && outCount === 1) {
    return {
      instructionName: "addLiquidityOneSide",
      instructionType: "add",
    };
  }

  // If we're sending and receiving tokens and can't match other patterns
  if (inCount > 0 && outCount > 0) {
    return {
      instructionName: "addLiquidityByStrategy",
      instructionType: "add",
    };
  }

  // Default to add liquidity if we can't determine a more specific type
  return {
    instructionName: "addLiquidity",
    instructionType: "add",
  };
}

/**
 * Extract Meteora instructions from transaction data
 */
export function extractMeteoraInstructions(
  transaction: ParsedTransactionWithMeta
): MeteoraDlmmInstruction[] {
  try {
    console.log(
      `==== Analyzing transaction: ${transaction.transaction.signatures[0]?.slice(0, 8)} ====`
    );

    if (!transaction.blockTime) {
      console.log("Skipping: No blockTime");
      return [];
    }

    // Log some basic info about the transaction
    console.log(
      `Block time: ${new Date(transaction.blockTime * 1000).toISOString()}`
    );
    console.log(`Signatures: ${transaction.transaction.signatures.join(", ")}`);
    console.log(
      `Account keys count: ${transaction.transaction.message.accountKeys.length}`
    );

    // Check if transaction is from Meteora program
    let hasMeteora = false;

    // Log all program IDs involved
    console.log("Programs in transaction:");
    const programIds = new Set<string>();

    // Check in instructions
    for (const ix of transaction.transaction.message.instructions) {
      if ("programId" in ix) {
        const programId = ix.programId.toString();
        programIds.add(programId);

        if (programId === METEORA_PROGRAM_ID) {
          hasMeteora = true;
          console.log(`✅ Found Meteora in main instructions: ${programId}`);
        }
      }
    }

    // If no Meteora instructions, check inner instructions
    if (!hasMeteora && transaction.meta?.innerInstructions) {
      console.log("Checking inner instructions:");
      for (const innerGroup of transaction.meta.innerInstructions) {
        for (const ix of innerGroup.instructions) {
          if ("programId" in ix) {
            const programId = ix.programId.toString();
            programIds.add(programId);

            if (programId === METEORA_PROGRAM_ID) {
              hasMeteora = true;
              console.log(
                `✅ Found Meteora in inner instructions: ${programId}`
              );
            }
          }
        }
      }
    }

    // Print all program IDs for debugging
    console.log("All program IDs in transaction:");
    for (const id of programIds) {
      console.log(` - ${id}${id === METEORA_PROGRAM_ID ? " (METEORA)" : ""}`);
    }

    // If no Meteora program ID found, skip this transaction
    if (!hasMeteora) {
      console.log("❌ No Meteora program found in transaction, skipping");
      return [];
    }

    // Log token balances if present
    if (
      transaction.meta?.preTokenBalances &&
      transaction.meta?.postTokenBalances
    ) {
      console.log("Token balances found:");
      console.log(
        `Pre token balances: ${transaction.meta.preTokenBalances.length}`
      );
      console.log(
        `Post token balances: ${transaction.meta.postTokenBalances.length}`
      );
    } else {
      console.log("No token balances found");
    }

    // Extract token transfers
    const tokenTransfers = extractTokenTransfers(transaction, 0);
    console.log(`Token transfers detected: ${tokenTransfers.length}`);

    for (const transfer of tokenTransfers) {
      console.log(
        ` - Mint: ${transfer.mint.slice(0, 8)}... Amount: ${transfer.amount} Direction: ${transfer.direction}`
      );
    }

    // If we have Meteora and token transfers, create a simplified instruction
    const signature = transaction.transaction.signatures[0] || "";

    // Try to identify instruction type based on token transfers
    const { instructionName, instructionType } =
      identifyInstructionType(tokenTransfers);
    console.log(
      `Identified instruction: ${instructionName} (${instructionType})`
    );

    // Create a simplified instruction with detected info
    const instruction: MeteoraDlmmInstruction = {
      signature,
      slot: transaction.slot,
      blockTime: transaction.blockTime,
      instructionName,
      // Get instructionType from the map if available, otherwise use the one from identification
      instructionType: INSTRUCTION_MAP[instructionName] || instructionType,
      accounts: {
        // We won't try to be too precise about position/pair/sender for now
        position: "detected",
        lbPair: "detected",
        sender: "detected",
      },
      tokenTransfers,
      activeBinId: null,
      removalBps: null,
    };

    console.log(`Created instruction: ${JSON.stringify(instruction, null, 2)}`);
    return [instruction];
  } catch (error) {
    console.error("Error parsing Meteora instructions:", error);
    return [];
  }
}

/**
 * Extract token transfers from a transaction
 */
function extractTokenTransfers(
  transaction: ParsedTransactionWithMeta,
  instructionIndex: number
): TokenTransferInfo[] {
  try {
    const transfers: TokenTransferInfo[] = [];

    // Check pre/post token balances to infer transfers
    const preBalances = transaction.meta?.preTokenBalances || [];
    const postBalances = transaction.meta?.postTokenBalances || [];

    console.log(
      `Analyzing token transfers: Pre: ${preBalances.length}, Post: ${postBalances.length}`
    );

    // First find all new token accounts (those that appear in post but not pre)
    const newAccounts = postBalances.filter(
      (post) =>
        !preBalances.some((pre) => pre.accountIndex === post.accountIndex)
    );

    if (newAccounts.length > 0) {
      console.log(`  Found ${newAccounts.length} newly created token accounts`);

      // For new accounts, consider them as receiving tokens
      for (const newAccount of newAccounts) {
        if (newAccount?.mint) {
          const amount = Number(newAccount.uiTokenAmount.amount) || 0;
          if (amount > 0) {
            transfers.push({
              mint: newAccount.mint,
              amount,
              direction: "in",
            });
            console.log(
              `  ✅ Added transfer for new account: IN ${amount} of ${newAccount.mint.slice(0, 8)}...`
            );
          }
        }
      }
    }

    // Process all post balances with matching pre balances
    for (let i = 0; i < postBalances.length; i++) {
      const post = postBalances[i];
      if (!post) {
        console.log(`  Skipping undefined post balance at index ${i}`);
        continue;
      }

      const pre = preBalances.find((b) => b.accountIndex === post.accountIndex);
      console.log(
        `  Account ${post.accountIndex}: ${pre ? "Found" : "Not found"} matching pre-balance`
      );

      // Skip if we've already handled this as a new account
      if (!pre) {
        console.log(
          "  Already processed as new account or missing pre balance, skipping"
        );
        continue;
      }

      if (pre && post.mint) {
        const preAmount = Number(pre.uiTokenAmount.amount) || 0;
        const postAmount = Number(post.uiTokenAmount.amount) || 0;
        const diff = postAmount - preAmount;

        console.log(
          `  Mint: ${post.mint.slice(0, 8)}..., Pre: ${preAmount}, Post: ${postAmount}, Diff: ${diff}`
        );

        if (diff !== 0) {
          transfers.push({
            mint: post.mint,
            amount: Math.abs(diff),
            direction: diff > 0 ? "in" : "out",
          });
          console.log(
            `  ✅ Added transfer: ${diff > 0 ? "IN" : "OUT"} ${Math.abs(diff)} of ${post.mint.slice(0, 8)}...`
          );
        } else {
          console.log("  No change in balance, skipping");
        }
      } else {
        console.log("  Missing mint, skipping");
      }
    }

    // Try to create pairs for each token (often in Meteora, tokens are swapped in pairs)
    const transfersByMint = new Map<string, TokenTransferInfo[]>();

    // Group transfers by mint
    for (const transfer of transfers) {
      if (!transfersByMint.has(transfer.mint)) {
        transfersByMint.set(transfer.mint, []);
      }
      transfersByMint.get(transfer.mint)?.push(transfer);
    }

    console.log(`Total transfers detected: ${transfers.length}`);
    return transfers;
  } catch (error) {
    console.error("Error extracting token transfers:", error);
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

    console.log(
      `Starting to process ${transactions.length} transactions for Meteora operations`
    );

    // Process transactions in batches of 20
    const batchSize = 20;
    const batches = Math.ceil(transactions.length / batchSize);
    console.log(`Processing in ${batches} batches of ${batchSize}`);

    const processNextBatch = async (batchIndex: number) => {
      if (batchIndex >= batches) {
        console.log(
          `✅ Completed processing all ${transactions.length} transactions`
        );
        console.log(
          `✅ Found ${meteoraInstructions.length} Meteora instructions`
        );
        console.log("Instruction types:");

        // Count by instruction type
        const counts: Record<MeteoraDlmmInstructionType, number> = {
          open: 0,
          add: 0,
          remove: 0,
          claim: 0,
          close: 0,
        };

        for (const ix of meteoraInstructions) {
          counts[ix.instructionType]++;
        }

        console.log(` - Open: ${counts.open}`);
        console.log(` - Add: ${counts.add}`);
        console.log(` - Remove: ${counts.remove}`);
        console.log(` - Claim: ${counts.claim}`);
        console.log(` - Close: ${counts.close}`);

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
      console.log(
        `Processing batch ${batchIndex + 1}/${batches} (Transactions ${start}-${end - 1})`
      );

      // Process each transaction in the batch
      for (const tx of batch) {
        try {
          console.log(
            `Fetching transaction ${tx.signature.slice(0, 8)}... (${processed + 1}/${transactions.length})`
          );
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
            console.log(
              `Successfully fetched transaction ${tx.signature.slice(0, 8)}...`
            );
            const instructions = extractMeteoraInstructions(
              jsonResponse.result
            );

            if (instructions.length > 0) {
              console.log(
                `✅ Found ${instructions.length} Meteora instructions in transaction ${tx.signature.slice(0, 8)}...`
              );
              meteoraInstructions.push(...instructions);
            } else {
              console.log(
                `❌ No Meteora instructions found in transaction ${tx.signature.slice(0, 8)}...`
              );
            }
          } else {
            console.log(
              `❌ Failed to fetch transaction ${tx.signature.slice(0, 8)}... Error: ${jsonResponse.error?.message || "Unknown error"}`
            );
          }
        } catch (error) {
          console.error(`Error processing transaction ${tx.signature}:`, error);
        }

        processed++;
        if (processed % 10 === 0 && onProgress) {
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
