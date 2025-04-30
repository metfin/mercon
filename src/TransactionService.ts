import {
  Connection,
  PublicKey,
  type ConfirmedSignatureInfo,
  type ParsedTransactionWithMeta,
} from "@solana/web3.js";
import type { TransactionData } from "../types";
import {
  METEORA_PROGRAM_ID,
  extractMeteoraInstructions,
  type MeteoraDlmmInstruction,
} from "./MeteoraParser";

export class TransactionService {
  private connection: Connection;
  private walletAddress: PublicKey;

  constructor(rpcUrl: string, walletAddress: string) {
    this.connection = new Connection(rpcUrl);
    this.walletAddress = new PublicKey(walletAddress);
  }

  /**
   * Helper to handle retries with exponential backoff for RPC requests
   * @param operation The async function to retry
   * @param maxRetries Maximum number of retry attempts
   * @param baseDelay Initial delay in milliseconds
   */
  private async withRetry<T>(
    operation: () => Promise<T>,
    maxRetries = 5,
    baseDelay = 1000,
    operationName = "RPC operation"
  ): Promise<T> {
    let retries = 0;

    while (true) {
      try {
        return await operation();
      } catch (error) {
        retries++;

        // Detailed error logging
        const errorMsg = error instanceof Error ? error.message : String(error);
        console.error(`${operationName} failed: ${errorMsg}`);

        // Check if this is a rate limit error
        const isRateLimit =
          error instanceof Error &&
          (errorMsg.includes("429") || errorMsg.includes("Too Many Requests"));

        // If we've reached max retries or error is not a rate limit issue, throw
        if (retries > maxRetries || !isRateLimit) {
          console.error(
            `${operationName} failed after ${retries} attempts, giving up.`
          );
          throw error;
        }

        // Calculate exponential backoff with jitter
        const delay = Math.min(
          baseDelay * 2 ** retries + Math.random() * 300,
          10000 // Max 10 second delay
        );

        console.log(
          `[RATE LIMIT] ${operationName} rate limited (attempt ${retries}/${maxRetries}). Retrying after ${Math.round(delay)}ms delay...`
        );
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }
  }

  /**
   * Helper to throttle requests to avoid rate limits
   * @param ms Time to wait in milliseconds
   */
  private async delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Fetch transaction signatures for the wallet
   * @param limit Maximum number of signatures to fetch
   * @param before Optional signature to fetch transactions before
   * @param until Optional signature to fetch transactions until
   */
  public async getTransactionSignatures(
    limit = 1000,
    before?: string,
    until?: string
  ): Promise<ConfirmedSignatureInfo[]> {
    console.log(
      `[SIGNATURES] Fetching ${limit} signatures${before ? ` before ${before.slice(0, 8)}...` : ""}`
    );

    return this.withRetry(
      async () => {
        try {
          const options: {
            limit: number;
            before?: string;
            until?: string;
          } = { limit };

          if (before) options.before = before;
          if (until) options.until = until;

          const result = await this.connection.getSignaturesForAddress(
            this.walletAddress,
            options
          );

          console.log(
            `[SIGNATURES] Successfully fetched ${result.length} signatures`
          );
          return result;
        } catch (error) {
          console.error("Error fetching transaction signatures:", error);
          throw new Error(
            `Failed to fetch transaction signatures: ${error instanceof Error ? error.message : String(error)}`
          );
        }
      },
      5,
      1000,
      "Signature fetch"
    );
  }

  /**
   * Fetch all signatures first, then process transactions in batches with progress updates
   * @param batchSize Size of each processing batch (max 300 due to RPC limitations)
   * @param onProgress Callback for progress updates
   */
  public async getTransactionsInBatches(
    batchSize = 300,
    onProgress?: (status: string, progress: number, total: number) => void
  ): Promise<TransactionData[]> {
    // Step 1: Fetch all signatures first
    const allSignatures: ConfirmedSignatureInfo[] = [];
    let before: string | undefined;
    let hasMore = true;

    onProgress?.("Fetching signatures", 0, 0);

    while (hasMore) {
      const signatures = await this.getTransactionSignatures(1000, before);

      // Add delay between signature fetches to avoid rate limits
      await this.delay(500);

      if (signatures.length === 0) {
        hasMore = false;
        break;
      }

      allSignatures.push(...signatures);
      onProgress?.(
        "Fetching signatures",
        allSignatures.length,
        signatures.length < 1000
          ? allSignatures.length
          : allSignatures.length + 1000
      );

      // Update the before parameter for the next batch
      if (signatures.length > 0) {
        before = signatures[signatures.length - 1]?.signature;
      }

      // If we got less than 1000 signatures, we've reached the end
      if (signatures.length < 1000) {
        hasMore = false;
      }
    }

    // Step 2: Process transactions in batches of 300
    const transactions: TransactionData[] = [];
    const signatureStrings = allSignatures.map((sig) => sig.signature);
    const totalBatches = Math.ceil(signatureStrings.length / batchSize);

    for (let i = 0; i < totalBatches; i++) {
      const batchSignatures = signatureStrings.slice(
        i * batchSize,
        (i + 1) * batchSize
      );

      onProgress?.("Processing transactions", i + 1, totalBatches);

      const batchTransactions = await this.getTransactions(batchSignatures);
      transactions.push(...batchTransactions);

      // Add delay between transaction batch fetches
      await this.delay(1000);
    }

    onProgress?.("Completed", totalBatches, totalBatches);

    return transactions;
  }

  /**
   * Process transactions concurrently with realtime analysis
   * @param meteoraProgramId The Meteora program ID to filter transactions
   * @param onTransactionProcessed Callback for when transactions are processed
   * @param onProgress Callback for progress updates
   * @param until Optional signature to fetch transactions until
   * @param maxSignatures Maximum number of signatures to process (0 for all)
   * @param batchSize Size of each processing batch (max 300 due to RPC limitations)
   */
  public async analyzeMeteoraBatches(
    meteoraProgramId: string = METEORA_PROGRAM_ID,
    onTransactionProcessed?: (
      transactions: TransactionData[],
      meteoraInstructions: MeteoraDlmmInstruction[]
    ) => void | Promise<void>,
    onProgress?: (status: string, progress: number, total: number) => void,
    until?: string,
    maxSignatures = 0,
    batchSize = 300
  ): Promise<MeteoraDlmmInstruction[]> {
    // First get all signatures
    const allSignatures: ConfirmedSignatureInfo[] = [];
    let before: string | undefined;
    let hasMore = true;

    onProgress?.("Fetching signatures", 0, 0);

    while (hasMore) {
      const signatures = await this.getTransactionSignatures(
        1000,
        before,
        until
      );

      // Add delay between signature requests
      await this.delay(500);

      if (signatures.length === 0) {
        hasMore = false;
        break;
      }

      allSignatures.push(...signatures);
      onProgress?.(
        "Fetching signatures",
        allSignatures.length,
        signatures.length < 1000
          ? allSignatures.length
          : allSignatures.length + 1000
      );

      // Update the before parameter for the next batch
      if (signatures.length > 0) {
        before = signatures[signatures.length - 1]?.signature;
      }

      // If we got less than 1000 signatures, we've reached the end
      if (signatures.length < 1000) {
        hasMore = false;
      }

      // Check if we've reached the maximum requested signatures
      if (maxSignatures > 0 && allSignatures.length >= maxSignatures) {
        hasMore = false;
      }
    }

    // Process transactions in concurrent batches and analyze as they come in
    const meteoraInstructions: MeteoraDlmmInstruction[] = [];
    const signatureStrings = allSignatures.map((sig) => sig.signature);
    const totalBatches = Math.ceil(signatureStrings.length / batchSize);

    for (let i = 0; i < totalBatches; i++) {
      const batchSignatures = signatureStrings.slice(
        i * batchSize,
        (i + 1) * batchSize
      );

      onProgress?.("Processing transactions", i + 1, totalBatches);

      const batchTransactions = await this.getTransactions(batchSignatures);

      // Add delay between transaction batch processing
      await this.delay(1000);

      // Process the transactions to find Meteora instructions
      const batchFullTransactions = await Promise.all(
        batchSignatures.map(async (signature) => {
          try {
            return await this.getTransaction(signature);
          } catch (error) {
            console.error(`Error fetching transaction ${signature}:`, error);
            return null;
          }
        })
      );

      // Extract Meteora instructions from transactions
      const batchMeteoraInstructions: MeteoraDlmmInstruction[] = [];
      for (const tx of batchFullTransactions) {
        if (tx) {
          const instructions = extractMeteoraInstructions(tx);
          batchMeteoraInstructions.push(...instructions);
        }
      }

      // Add to the overall collection
      meteoraInstructions.push(...batchMeteoraInstructions);

      // If callback is provided, process transactions as they come in
      if (onTransactionProcessed) {
        onTransactionProcessed(batchTransactions, batchMeteoraInstructions);
      }

      // Add delay between batches
      await this.delay(1000);
    }

    onProgress?.("Completed", totalBatches, totalBatches);

    return meteoraInstructions;
  }

  /**
   * Fetch all transactions for the wallet with optional pagination
   * @param maxTransactions Maximum number of transactions to fetch (0 for all)
   * @param onProgress Optional progress callback
   */
  public async fetchAllTransactions(
    maxTransactions = 0
  ): Promise<TransactionData[]> {
    const batchSize = 300;
    let allTransactions: TransactionData[] = [];
    let before: string | undefined;
    let keepFetching = true;
    let fetchedCount = 0;

    while (keepFetching) {
      const signatures = await this.getTransactionSignatures(batchSize, before);

      // Add delay between signature batches
      await this.delay(500);

      if (signatures.length === 0) {
        keepFetching = false;
        break;
      }

      const signatureStrings = signatures.map((sig) => sig.signature);
      const transactions = await this.getTransactions(signatureStrings);

      // Add delay between transaction batches
      await this.delay(1000);

      allTransactions = [...allTransactions, ...transactions];
      fetchedCount += signatures.length;

      // Update the before parameter for the next batch
      if (signatures.length > 0) {
        before = signatures[signatures.length - 1]?.signature;
      }

      // Check if we've reached the maximum requested transactions
      if (maxTransactions > 0 && fetchedCount >= maxTransactions) {
        keepFetching = false;
      }
    }
    return allTransactions;
  }
}
