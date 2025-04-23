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
    try {
      const options: {
        limit: number;
        before?: string;
        until?: string;
      } = { limit };

      if (before) options.before = before;
      if (until) options.until = until;

      return await this.connection.getSignaturesForAddress(
        this.walletAddress,
        options
      );
    } catch (error) {
      console.error("Error fetching transaction signatures:", error);
      throw new Error(
        `Failed to fetch transaction signatures: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  /**
   * Fetch transaction details for a given signature
   * @param signature Transaction signature
   */
  public async getTransaction(
    signature: string
  ): Promise<ParsedTransactionWithMeta | null> {
    try {
      return await this.connection.getParsedTransaction(signature, {
        maxSupportedTransactionVersion: 0,
      });
    } catch (error) {
      console.error(`Error fetching transaction ${signature}:`, error);
      throw new Error(
        `Failed to fetch transaction ${signature}: ${error instanceof Error ? error.message : String(error)}`
      );
    }
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
    }

    onProgress?.("Completed", totalBatches, totalBatches);

    return transactions;
  }

  /**
   * Fetch multiple transactions and process them
   * @param signatures List of transaction signatures
   */
  public async getTransactions(
    signatures: string[]
  ): Promise<TransactionData[]> {
    const transactions: TransactionData[] = [];

    try {
      // Create batch requests using the RPC endpoint directly
      const requests = signatures.map((signature, i) => ({
        method: "getTransaction",
        params: [
          signature,
          { encoding: "jsonParsed", maxSupportedTransactionVersion: 0 },
        ],
        id: `${i}`,
        jsonrpc: "2.0",
      }));

      // Get the RPC endpoint from the connection
      const rpcUrl = this.connection.rpcEndpoint;

      // Send the batch request
      const response = await fetch(rpcUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requests),
      });

      const results = await response.json();

      // Process results
      if (Array.isArray(results)) {
        for (let i = 0; i < results.length; i++) {
          const result = results[i];
          const signature = signatures[i];

          if (signature && result && result.result) {
            const tx = result.result;
            if (tx?.blockTime) {
              transactions.push({
                signature,
                timestamp: tx.blockTime * 1000, // Convert to milliseconds
                blockTime: tx.blockTime,
                slot: tx.slot,
                status: tx.meta?.err ? "failed" : "success",
              });
            }
          }
        }
      }
    } catch (error) {
      console.error("Error processing batch transactions:", error);
    }

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

      if (signatures.length === 0) {
        keepFetching = false;
        break;
      }

      const signatureStrings = signatures.map((sig) => sig.signature);
      const transactions = await this.getTransactions(signatureStrings);

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
