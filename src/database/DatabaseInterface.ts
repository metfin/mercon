import type { MeteoraDlmmInstruction } from "../services/MeteoraParser";
import type {
  MeteoraDlmmPairData,
  MeteoraPositionTransactions,
  MeteoraDlmmDbTransactions,
} from "../types/meteora";
import type { TokenMeta } from "../types/tokens";

/**
 * Interface for all database operations
 * This provides a common abstraction for different database implementations
 */
export interface DatabaseInterface {
  // Initialize the database
  initialize(): Promise<void>;

  // Close the database connection
  close(): Promise<void>;

  // Save current state to persistent storage
  save(): Promise<void>;

  // Instruction operations
  addInstruction(instruction: MeteoraDlmmInstruction): Promise<void>;
  addTransfers(instruction: MeteoraDlmmInstruction): Promise<void>;

  // Pair operations
  addPair(pair: MeteoraDlmmPairData): Promise<void>;
  getLbPair(positionAddress: string): Promise<string | undefined>;
  getMissingPairs(): Promise<string[]>;

  // Token operations
  addToken(token: TokenMeta): Promise<void>;
  getMissingTokens(): Promise<string[]>;

  // USD operations
  addUsdTransactions(
    positionAddress: string,
    transactions: MeteoraPositionTransactions
  ): Promise<void>;
  getMissingUsd(): Promise<string[]>;

  // Position operations
  getPositionTransactions(
    positionAddress: string
  ): Promise<MeteoraPositionTransactions | null>;

  // Account operations
  markComplete(accountAddress: string): Promise<void>;
  isComplete(accountAddress: string): Promise<boolean>;
  setOldestSignature(
    accountAddress: string,
    oldestBlockTime: number,
    oldestSignature: string
  ): Promise<void>;

  // Transaction/signature operations
  getMostRecentSignature(ownerAddress: string): Promise<string | undefined>;
  getOldestSignature(ownerAddress: string): Promise<string | undefined>;
  addTransaction(transaction: {
    signature: string;
    owner: string;
    timestamp: string;
    slot: number;
  }): Promise<void>;

  // Get all transactions
  getAllTransactions(): Promise<MeteoraDlmmDbTransactions[]>;
  getOwnerTransactions(
    ownerAddress: string
  ): Promise<MeteoraDlmmDbTransactions[]>;

  // Wait for pending operations to complete
  waitForSave(): Promise<void>;
}
