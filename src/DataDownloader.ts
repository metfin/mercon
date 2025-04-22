import { Config } from "./config";
import { TransactionService } from "./services/TransactionService";
import { MeteoraService } from "./services/MeteoraService";
import { TokenPriceService } from "./services/TokenPriceService";
import type { DataDownloaderConfig, DownloadedData } from "./types";

export class DataDownloader {
	private config: Config;
	private transactionService: TransactionService;
	private meteoraService: MeteoraService;
	private tokenPriceService: TokenPriceService;

	constructor(config: DataDownloaderConfig) {
		this.config = new Config(config);
		this.transactionService = new TransactionService(
			this.config.getRpcUrl(),
			this.config.getWalletAddress(),
		);
		this.meteoraService = new MeteoraService();
		this.tokenPriceService = new TokenPriceService();
	}

	/**
	 * Start downloading data
	 * @param maxTransactions Maximum number of transactions to fetch (0 for all)
	 */
	public async download(maxTransactions = 1000): Promise<DownloadedData> {
		const callbacks = this.config.getCallbacks();
		const data: DownloadedData = {};

		try {
			// Fetch transaction data
			callbacks.onProgress?.(0, "Starting data download...");
			data.transactions =
				await this.transactionService.fetchAllTransactions(maxTransactions);
			callbacks.onProgress?.(
				40,
				`Fetched ${data.transactions.length} transactions`,
			);

			callbacks.onProgress?.(100, "Data download completed");

			// Call the onDone callback
			callbacks.onDone?.(data);

			return data;
		} catch (error) {
			console.error("Error downloading data:", error);
			callbacks.onError?.(
				error instanceof Error ? error : new Error(String(error)),
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
