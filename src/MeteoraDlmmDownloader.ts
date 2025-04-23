import type { DatabaseInterface } from "./database/DatabaseInterface";
import { TransactionService } from "./services/TransactionService";
import { MeteoraService } from "./services/MeteoraService";
import { METEORA_PROGRAM_ID } from "./services/MeteoraParser";
import type { MeteoraDlmmInstruction } from "./services/MeteoraParser";

export interface DownloaderCallbacks {
  onProgress?: (status: string, current: number, total: number) => void;
  onDone?: () => void;
  onError?: (error: Error) => void;
}

export interface MeteoraDownloaderConfig {
  account: string;
  rpcUrl: string;
  callbacks?: DownloaderCallbacks;
  batchSize?: number;
  maxTransactions?: number;
}

/**
 * Status of the downloader
 */
export interface DownloaderStatus {
  isRunning: boolean;
  isCancelled: boolean;
  progress: {
    current: number;
    total: number;
    percentage: number;
  };
  status: string;
  account: string;
  lastUpdated: Date;
}

/**
 * Downloader for Meteora DLMM data
 */
export class MeteoraDlmmDownloader {
  private db: DatabaseInterface;
  private transactionService: TransactionService;
  private meteoraService: MeteoraService;
  private config: MeteoraDownloaderConfig;

  private isRunning = false;
  private isCancelled = false;
  private progress = { current: 0, total: 0, percentage: 0 };
  private status = "idle";
  private lastUpdated = new Date();
  private downloadPromise: Promise<void> | null = null;

  /**
   * Create a new Meteora DLMM downloader
   */
  constructor(db: DatabaseInterface, config: MeteoraDownloaderConfig) {
    this.db = db;
    this.config = {
      ...config,
      batchSize: config.batchSize || 300,
      maxTransactions: config.maxTransactions || 0,
    };

    this.transactionService = new TransactionService(
      config.rpcUrl,
      config.account
    );

    this.meteoraService = new MeteoraService();
  }

  /**
   * Start downloading data
   */
  public start(): Promise<void> {
    if (this.isRunning) {
      throw new Error("Download already in progress");
    }

    this.isRunning = true;
    this.isCancelled = false;
    this.updateStatus("Starting download");

    this.downloadPromise = this.download();
    return this.downloadPromise;
  }

  /**
   * Cancel the download
   */
  public cancel(): void {
    if (!this.isRunning) return;

    this.isCancelled = true;
    this.updateStatus("Cancelling download");
  }

  /**
   * Get the current status of the downloader
   */
  public getStatus(): DownloaderStatus {
    return {
      isRunning: this.isRunning,
      isCancelled: this.isCancelled,
      progress: { ...this.progress },
      status: this.status,
      account: this.config.account,
      lastUpdated: new Date(this.lastUpdated.getTime()),
    };
  }

  /**
   * Update the current status
   */
  private updateStatus(status: string, current = 0, total = 0): void {
    this.status = status;

    if (total > 0) {
      this.progress.current = current;
      this.progress.total = total;
      this.progress.percentage = Math.round((current / total) * 100);
    }

    this.lastUpdated = new Date();

    // Call the progress callback if available
    this.config.callbacks?.onProgress?.(status, current, total);
  }

  /**
   * Download Meteora DLMM data
   */
  private async download(): Promise<void> {
    try {
      const account = this.config.account;

      // Check if account is already completed
      const isComplete = await this.db.isComplete(account);
      if (isComplete) {
        this.updateStatus("Account already completed", 100, 100);
        this.isRunning = false;
        this.config.callbacks?.onDone?.();
        return;
      }

      // Get most recent signature to use as a starting point
      const mostRecentSignature = await this.db.getMostRecentSignature(account);

      // Get the signatures in batches
      this.updateStatus("Fetching transaction signatures", 0, 0);

      // Get instructions from transactions
      await this.processTransactions(mostRecentSignature || undefined);

      // Find and add missing pairs
      await this.processMissingPairs();

      // Find and add missing tokens
      await this.processMissingTokens();

      // Find and add missing USD data
      await this.processMissingUsd();

      // Mark account as complete
      if (!this.isCancelled) {
        await this.db.markComplete(account);
        this.updateStatus("Download completed", 100, 100);
      } else {
        this.updateStatus(
          "Download cancelled",
          this.progress.current,
          this.progress.total
        );
      }

      // Save changes
      await this.db.save();

      // Call done callback
      this.isRunning = false;
      this.config.callbacks?.onDone?.();
    } catch (error) {
      this.isRunning = false;
      const err = error instanceof Error ? error : new Error(String(error));
      this.updateStatus(`Error: ${err.message}`, 0, 0);

      // Call error callback
      this.config.callbacks?.onError?.(err);

      throw err;
    }
  }

  /**
   * Process transactions to extract Meteora instructions
   */
  private async processTransactions(
    until?: string
  ): Promise<MeteoraDlmmInstruction[]> {
    // Analyze batches of Meteora transactions
    const instructions = await this.transactionService.analyzeMeteoraBatches(
      METEORA_PROGRAM_ID,
      async (_, instructions) => {
        if (this.isCancelled) return;

        // Process each instruction
        for (const instruction of instructions) {
          await this.db.addInstruction(instruction);
          await this.db.addTransfers(instruction);
        }
      },
      (status, current, total) => {
        if (this.isCancelled) return;

        this.updateStatus(`Analyzing transactions: ${status}`, current, total);
      },
      until,
      this.config.maxTransactions,
      this.config.batchSize
    );

    return instructions;
  }

  /**
   * Process missing pairs
   */
  private async processMissingPairs(): Promise<void> {
    if (this.isCancelled) return;

    this.updateStatus("Finding missing pairs", 0, 0);

    const missingPairs = await this.db.getMissingPairs();

    if (missingPairs.length === 0) {
      this.updateStatus("No missing pairs found", 0, 0);
      return;
    }

    this.updateStatus("Processing missing pairs", 0, missingPairs.length);

    for (let i = 0; i < missingPairs.length; i++) {
      if (this.isCancelled) break;

      const pairAddress = missingPairs[i];
      if (!pairAddress) continue;

      try {
        const pair = await this.meteoraService.getPair(pairAddress);

        if (pair) {
          await this.db.addPair({
            lbPair: pair.address,
            name: pair.name,
            mintX: pair.mint_x,
            mintY: pair.mint_y,
            binStep: pair.bin_step,
            baseFeeBps: Number.parseInt(pair.base_fee_percentage),
          });
        }

        this.updateStatus(
          "Processing missing pairs",
          i + 1,
          missingPairs.length
        );
      } catch (error) {
        console.error(`Error fetching pair ${pairAddress}:`, error);
      }
    }
  }

  /**
   * Process missing tokens
   */
  private async processMissingTokens(): Promise<void> {
    if (this.isCancelled) return;

    this.updateStatus("Finding missing tokens", 0, 0);

    const missingTokens = await this.db.getMissingTokens();

    if (missingTokens.length === 0) {
      this.updateStatus("No missing tokens found", 0, 0);
      return;
    }

    this.updateStatus("Processing missing tokens", 0, missingTokens.length);

    // In a real implementation, you would fetch token metadata from Jupiter or another source
    // For now, we'll add placeholder data
    for (let i = 0; i < missingTokens.length; i++) {
      if (this.isCancelled) break;

      const tokenAddress = missingTokens[i];
      if (!tokenAddress) continue;

      try {
        // This would be a real token lookup in production
        await this.db.addToken({
          address: tokenAddress,
          name: `Token ${tokenAddress.substring(0, 6)}`,
          symbol: `TKN${tokenAddress.substring(0, 3)}`,
          decimals: 9,
        });

        this.updateStatus(
          "Processing missing tokens",
          i + 1,
          missingTokens.length
        );
      } catch (error) {
        console.error(`Error adding token ${tokenAddress}:`, error);
      }
    }
  }

  /**
   * Process missing USD data
   */
  private async processMissingUsd(): Promise<void> {
    if (this.isCancelled) return;

    this.updateStatus("Finding positions missing USD data", 0, 0);

    const missingUsd = await this.db.getMissingUsd();

    if (missingUsd.length === 0) {
      this.updateStatus("No positions missing USD data", 0, 0);
      return;
    }

    this.updateStatus(
      "Processing positions for USD data",
      0,
      missingUsd.length
    );

    for (let i = 0; i < missingUsd.length; i++) {
      if (this.isCancelled) break;

      const positionAddress = missingUsd[i];
      if (!positionAddress) continue;

      try {
        // This would be a real API call in production
        // For now, we'll just mark all as filled with zero values
        await this.db.addUsdTransactions(positionAddress, {
          deposits: [],
          withdrawals: [],
          fees: [],
        });

        this.updateStatus(
          "Processing positions for USD data",
          i + 1,
          missingUsd.length
        );
      } catch (error) {
        console.error(
          `Error adding USD data for position ${positionAddress}:`,
          error
        );
      }
    }
  }
}
