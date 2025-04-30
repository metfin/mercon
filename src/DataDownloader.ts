import { Config } from "./config";
import { TransactionService } from "./services/TransactionService";
import { MeteoraService } from "./services/MeteoraService";
import { TokenPriceService } from "./services/TokenPriceService";
import type { DataDownloaderConfig } from "./types";
import type { DownloadedData } from "./types/downloaded-data";

export class DataDownloader {
  private config: Config;
  private transactionService: TransactionService;
  private meteoraService: MeteoraService;
  private tokenPriceService: TokenPriceService;

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
   * Start downloading data
   * @param maxTransactions Maximum number of transactions to fetch (0 for all)
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
      // Initial progress update
      this.reportProgress(callbacks.onProgress, 0, "INITIALIZING", 0, 0);

      // Step 1: Download transaction data with progress updates
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

      // Step 2: Analyze Meteora transactions (if we had any)
      this.reportProgress(
        callbacks.onProgress,
        75,
        "ANALYZING_METEORA",
        0,
        100
      );

      // Here we would analyze Meteora data
      // For now, we'll just simulate progress
      for (let i = 0; i <= 100; i += 10) {
        // Don't report too frequently to avoid console spam
        if (i % 20 === 0) {
          this.reportProgress(
            callbacks.onProgress,
            75 + (i / 100) * 15,
            "ANALYZING_METEORA",
            i,
            100
          );
          // Small delay for simulation
          await new Promise((resolve) => setTimeout(resolve, 100));
        }
      }

      // Step 3: Update token prices (if needed)
      this.reportProgress(callbacks.onProgress, 90, "UPDATING_PRICES", 0, 100);

      // Here we would update token prices
      // For now, we'll just simulate progress
      for (let i = 0; i <= 100; i += 20) {
        this.reportProgress(
          callbacks.onProgress,
          90 + (i / 100) * 10,
          "UPDATING_PRICES",
          i,
          100
        );
        // Small delay for simulation
        await new Promise((resolve) => setTimeout(resolve, 50));
      }

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
