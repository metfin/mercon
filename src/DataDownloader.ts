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
      // Fetch transaction data
      callbacks.onProgress?.(0, 100);

      // Step 1: Download transaction data with progress updates
      callbacks.onProgress?.(5, 100);

      const transactionsData =
        await this.transactionService.getTransactionsInBatches(
          300, // batch size
          (status, current, total) => {
            // Map progress from 5% to 40%
            const progressPercent = 5 + (current / total) * 35;
            callbacks.onProgress?.(progressPercent, 100);
          }
        );

      // Update transaction count in data
      data.transactions = transactionsData.length;

      callbacks.onProgress?.(40, 100);

      // Step 2: Analyze Meteora transactions
      callbacks.onProgress?.(45, 100);

      // Update end time
      data.endTime = new Date();

      callbacks.onProgress?.(90, 100);

      callbacks.onProgress?.(100, 100);

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
   * Create a new DataDownloader instance from environment variables
   */
  public static fromEnv(): DataDownloader {
    const config = Config.createFromEnv();
    return new DataDownloader(config.getConfig());
  }
}
