import { Config } from "./config";
import { TransactionService } from "./services/TransactionService";
import { MeteoraService } from "./services/MeteoraService";
import { TokenPriceService } from "./services/TokenPriceService";
import type { DataDownloaderConfig } from "./types";
import type { DownloadedData } from "./types/downloaded-data";
import { createDatabase, DatabaseType } from "./database/DatabaseFactory";
import type { DatabaseInterface } from "./database/DatabaseInterface";
import type { TransactionData } from "./types";
import {
  extractMeteoraInstructions,
  type MeteoraDlmmInstruction,
} from "./services/MeteoraParser";
import type { MeteoraDlmmPairData } from "./types/meteora";

export class DataDownloader {
  private config: Config;
  private transactionService: TransactionService;
  private meteoraService: MeteoraService;
  private tokenPriceService: TokenPriceService;
  private db!: DatabaseInterface; // Using definite assignment assertion
  private isCancelled = false;

  constructor(config: DataDownloaderConfig) {
    this.config = new Config(config);
    this.transactionService = new TransactionService(
      this.config.getRpcUrl(),
      this.config.getWalletAddress()
    );
    this.meteoraService = new MeteoraService();
    this.tokenPriceService = new TokenPriceService();
  }

  /**
   * Initialize database connection
   */
  private async initializeDatabase(): Promise<void> {
    // Create a SQLite database instance
    this.db = await createDatabase({
      type: DatabaseType.SQLITE,
      path: `./data/${this.config.getWalletAddress()}.db`,
    });

    await this.db.initialize();
  }

  /**
   * Cancel the download process
   */
  public cancel(): void {
    this.isCancelled = true;
  }

  /**
   * Start downloading data with better type handling
   */
  public async download(): Promise<DownloadedData> {
    const callbacks = {
      onDone: this.config.getOnDone(),
      onProgress: this.config.getOnProgress(),
      onError: this.config.getOnError(),
    };
    const startTime = new Date();

    // Initialize data with required properties from DownloadedData interface
    const data: DownloadedData = {
      account: this.config.getWalletAddress(),
      pairs: [],
      positions: [],
      transactions: 0,
      startTime,
      endTime: startTime, // Will be updated at the end
    };

    try {
      // Initialize database first
      await this.initializeDatabase();

      // Initial progress update
      this.reportProgress(callbacks.onProgress, 0, "INITIALIZING", 0, 0);

      // Step 1: Download transaction data with progress updates
      this.reportProgress(callbacks.onProgress, 5, "FETCHING_SIGNATURES", 0, 0);

      // Check if wallet is already processed
      const isComplete = await this.db.isComplete(
        this.config.getWalletAddress()
      );
      if (isComplete) {
        console.log(
          `Wallet ${this.config.getWalletAddress()} already processed. Skipping.`
        );
        this.reportProgress(callbacks.onProgress, 100, "COMPLETED", 100, 100);
        data.endTime = new Date();
        callbacks.onDone?.(data);
        return data;
      }

      // Fetch transactions in batches with progress updates
      const transactionsData =
        await this.transactionService.getTransactionsInBatches(
          300, // batch size
          (status, current, total) => {
            // Map progress from 0% to 70%
            let progressPercent: number;
            let statusType: string;

            if (status === "Fetching signatures") {
              statusType = "FETCHING_SIGNATURES";
              progressPercent = (current / (total || current * 2)) * 35;
            } else if (status === "Processing transactions") {
              statusType = "PROCESSING_TRANSACTIONS";
              progressPercent = 35 + (current / total) * 35;
            } else {
              statusType = status.toUpperCase().replace(/ /g, "_");
              progressPercent = 70;
            }

            this.reportProgress(
              callbacks.onProgress,
              progressPercent,
              statusType,
              current,
              total
            );
          }
        );

      // Update transaction count in data
      data.transactions = transactionsData.length;

      // Save all transactions to database
      this.reportProgress(
        callbacks.onProgress,
        70,
        "SAVING_TRANSACTIONS",
        0,
        transactionsData.length
      );
      for (let i = 0; i < transactionsData.length; i++) {
        if (this.isCancelled) break;

        const tx = transactionsData[i];
        if (tx) {
          await this.db.addTransaction({
            signature: tx.signature,
            owner: this.config.getWalletAddress(),
            timestamp: new Date(tx.blockTime * 1000).toISOString(),
            slot: tx.slot,
          });
        }

        if (i % 50 === 0) {
          this.reportProgress(
            callbacks.onProgress,
            70 + (i / transactionsData.length) * 5,
            "SAVING_TRANSACTIONS",
            i,
            transactionsData.length
          );
        }
      }

      // Step 2: Analyze Meteora transactions
      if (!this.isCancelled) {
        this.reportProgress(
          callbacks.onProgress,
          75,
          "ANALYZING_METEORA",
          0,
          100
        );

        // Process Meteora instructions from transactions
        const meteoraInstructions =
          await this.processMeteoraBatches(transactionsData);

        // Extract unique positions and pairs
        const positionSet = new Set<string>();
        const pairSet = new Set<string>();

        for (const instruction of meteoraInstructions) {
          if (
            instruction.accounts?.position &&
            instruction.accounts.position !== "unknown"
          ) {
            positionSet.add(instruction.accounts.position);
          }
          if (
            instruction.accounts?.lbPair &&
            instruction.accounts.lbPair !== "unknown"
          ) {
            pairSet.add(instruction.accounts.lbPair);
          }
        }

        // Add to result data
        data.positions = Array.from(positionSet);
        data.pairs = Array.from(pairSet);
      }

      // Step 3: Process all position accounts
      if (!this.isCancelled && data.positions.length > 0) {
        this.reportProgress(
          callbacks.onProgress,
          85,
          "PROCESSING_POSITIONS",
          0,
          data.positions.length
        );

        // Fetch position data for each position
        for (let i = 0; i < data.positions.length; i++) {
          if (this.isCancelled) break;

          const positionAddress = data.positions[i];
          if (positionAddress) {
            await this.processPositionData(positionAddress);
          }

          this.reportProgress(
            callbacks.onProgress,
            85 + (i / data.positions.length) * 5,
            "PROCESSING_POSITIONS",
            i,
            data.positions.length
          );
        }
      }

      // Step 4: Process all pair accounts
      if (!this.isCancelled && data.pairs.length > 0) {
        this.reportProgress(
          callbacks.onProgress,
          90,
          "PROCESSING_PAIRS",
          0,
          data.pairs.length
        );

        // Fetch pair data for each pair
        for (let i = 0; i < data.pairs.length; i++) {
          if (this.isCancelled) break;

          const pairAddress = data.pairs[i];
          if (pairAddress) {
            await this.processPairData(pairAddress);
          }

          this.reportProgress(
            callbacks.onProgress,
            90 + (i / data.pairs.length) * 5,
            "PROCESSING_PAIRS",
            i,
            data.pairs.length
          );
        }
      }

      // Step 5: Update token prices
      if (!this.isCancelled) {
        this.reportProgress(
          callbacks.onProgress,
          95,
          "UPDATING_PRICES",
          0,
          100
        );

        // Get all missing tokens that need prices
        const missingTokens = await this.db.getMissingTokens();

        // Fetch token prices for each token
        for (let i = 0; i < missingTokens.length; i++) {
          if (this.isCancelled) break;

          const tokenAddress = missingTokens[i];
          if (tokenAddress) {
            await this.processTokenData(tokenAddress);
          }

          this.reportProgress(
            callbacks.onProgress,
            95 + (i / missingTokens.length) * 5,
            "UPDATING_PRICES",
            i,
            missingTokens.length
          );
        }
      }

      // Mark wallet as completely processed
      if (!this.isCancelled) {
        await this.db.markComplete(this.config.getWalletAddress());
      }

      // Save changes to database
      await this.db.save();

      // Final progress update
      this.reportProgress(callbacks.onProgress, 100, "COMPLETED", 100, 100);

      // Update end time
      data.endTime = new Date();

      // Call the onDone callback
      callbacks.onDone?.(data);

      return data;
    } catch (error) {
      console.error("Error downloading data:", error);
      callbacks.onError?.(
        error instanceof Error ? error : new Error(String(error))
      );
      throw error;
    } finally {
      // Close database connection
      if (this.db) {
        await this.db.save();
        await this.db.close();
      }
    }
  }

  /**
   * Process Meteora transactions in batches
   */
  private async processMeteoraBatches(
    transactions: TransactionData[]
  ): Promise<MeteoraDlmmInstruction[]> {
    const allInstructions: MeteoraDlmmInstruction[] = [];
    const batchSize = 50; // Process 50 transactions at a time
    const totalBatches = Math.ceil(transactions.length / batchSize);

    for (let i = 0; i < totalBatches; i++) {
      if (this.isCancelled) break;

      const start = i * batchSize;
      const end = Math.min(start + batchSize, transactions.length);
      const batchTransactions = transactions.slice(start, end);

      // Fetch full transaction data for signatures
      const signatures = batchTransactions.map((tx) => tx.signature);
      const fullTransactions =
        await this.transactionService.getTransactions(signatures);

      // Process each transaction to extract Meteora instructions
      for (const tx of fullTransactions) {
        if (this.isCancelled) break;

        try {
          // In a real implementation, you would fetch the full transaction data
          // and properly parse it using the Meteora parser
          const parsedTransaction =
            await this.transactionService.getTransaction(tx.signature);

          if (parsedTransaction) {
            // Extract Meteora instructions
            const instructions = extractMeteoraInstructions(parsedTransaction);

            // Save each instruction to the database
            for (const instruction of instructions) {
              if (this.isCancelled) break;

              // Save instructions to database - in a real implementation,
              // this would properly format the instruction data
              await this.db.addInstruction(instruction);

              // Also save token transfers
              if (
                instruction.tokenTransfers &&
                instruction.tokenTransfers.length > 0
              ) {
                await this.db.addTransfers(instruction);
              }

              allInstructions.push(instruction);
            }
          }
        } catch (error) {
          console.error(`Error processing transaction ${tx.signature}:`, error);
          // Continue with next transaction
        }
      }

      // Report progress
      this.reportProgress(
        this.config.getOnProgress(),
        75 + (i / totalBatches) * 10,
        "ANALYZING_METEORA",
        i + 1,
        totalBatches
      );
    }

    return allInstructions;
  }

  /**
   * Process position data with proper error handling
   */
  private async processPositionData(positionAddress: string): Promise<void> {
    if (!positionAddress) {
      console.warn("Skipping empty position address");
      return;
    }

    try {
      console.log(`Processing position ${positionAddress.slice(0, 8)}...`);

      // Fetch position data from Meteora API
      const positionData =
        await this.meteoraService.getPosition(positionAddress);

      if (!positionData) {
        console.warn(
          `Position ${positionAddress.slice(0, 8)} not found in Meteora API`
        );
        return;
      }

      // Get the pair for this position
      const pairAddress = positionData.pair_address;

      if (!pairAddress) {
        console.warn(
          `No pair address found for position ${positionAddress.slice(0, 8)}`
        );
        return;
      }

      // Process transactions for this position
      await this.processPairData(pairAddress);

      // Fetch position transactions from API
      const transactions =
        await this.meteoraService.getPositionTransactions(positionAddress);

      // Save position transactions to database
      if (transactions) {
        await this.db.addUsdTransactions(positionAddress, transactions);
        console.log(
          `Saved transactions for position ${positionAddress.slice(0, 8)}`
        );
      }
    } catch (error) {
      console.error(`Error processing position ${positionAddress}:`, error);
      // Continue with next position - this implements error recovery
    }
  }

  /**
   * Process pair data with proper error handling
   */
  private async processPairData(pairAddress: string): Promise<void> {
    if (!pairAddress) {
      console.warn("Skipping empty pair address");
      return;
    }

    try {
      console.log(`Processing pair ${pairAddress.slice(0, 8)}...`);

      // Fetch pair data from Meteora API
      const pairData = await this.meteoraService.getPair(pairAddress);

      if (!pairData) {
        console.warn(
          `Pair ${pairAddress.slice(0, 8)} not found in Meteora API`
        );
        return;
      }

      // Map API data to our database format
      const mintX = pairData.mint_x || "";
      const mintY = pairData.mint_y || "";

      const mappedPairData = {
        lbPair: pairAddress,
        name: pairData.name,
        mintX,
        mintY,
        binStep: pairData.bin_step,
        baseFeeBps: Number.parseInt(pairData.base_fee_percentage),
      };

      // Add pair data to database
      await this.db.addPair(mappedPairData as unknown as MeteoraDlmmPairData);

      // Also process the token data for this pair
      if (mintX) {
        await this.processTokenData(mintX);
      }

      if (mintY) {
        await this.processTokenData(mintY);
      }

      console.log(`Saved pair data for ${pairAddress.slice(0, 8)}`);
    } catch (error) {
      console.error(`Error processing pair ${pairAddress}:`, error);
      // Continue with next pair - this implements error recovery
    }
  }

  /**
   * Process token data with proper error handling
   */
  private async processTokenData(tokenAddress: string): Promise<void> {
    if (!tokenAddress) {
      console.warn("Skipping empty token address");
      return;
    }

    try {
      console.log(`Processing token ${tokenAddress.slice(0, 8)}...`);

      // Check if token already exists in database
      const missingTokens = await this.db.getMissingTokens();
      if (!missingTokens.includes(tokenAddress)) {
        console.log(
          `Token ${tokenAddress.slice(0, 8)} already in database, skipping`
        );
        return;
      }

      // Fetch token data from token price service
      const tokenData = await this.tokenPriceService.getTokenData(tokenAddress);

      if (!tokenData) {
        console.warn(
          `Token ${tokenAddress.slice(0, 8)} not found in token price API`
        );
        return;
      }

      // Add token data to database
      await this.db.addToken(tokenData);
      console.log(`Saved token data for ${tokenAddress.slice(0, 8)}`);
    } catch (error) {
      console.error(`Error processing token ${tokenAddress}:`, error);
      // Continue with next token - this implements error recovery
    }
  }

  /**
   * Helper to report progress with the correct format
   */
  private reportProgress(
    onProgress:
      | ((
          progress: number,
          total: number,
          statusType: string,
          current: number,
          totalItems: number
        ) => void)
      | undefined,
    progress: number,
    statusType: string,
    current: number,
    total: number
  ): void {
    if (onProgress) {
      onProgress(progress, 100, statusType, current, total);
    }
  }

  /**
   * Create a new DataDownloader instance from environment variables
   */
  public static fromEnv(): DataDownloader {
    const config = Config.createFromEnv();
    return new DataDownloader(config.getConfig());
  }
}
