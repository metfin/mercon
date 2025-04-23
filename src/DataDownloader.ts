import { Config } from "./config";
import { TransactionService } from "./services/TransactionService";
import { MeteoraService } from "./services/MeteoraService";
import { TokenPriceService } from "./services/TokenPriceService";
import { METEORA_PROGRAM_ID } from "./services/MeteoraParser";
import type { DataDownloaderConfig, DownloadedData } from "./types";
import type { MeteoraDlmmInstruction } from "./services/MeteoraParser";

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

			// Step 1: Download transaction data with progress updates
			callbacks.onProgress?.(5, "Downloading transaction data...");

			data.transactions =
				await this.transactionService.getTransactionsInBatches(
					300, // batch size
					(status, current, total) => {
						// Map progress from 5% to 40%
						const progressPercent = 5 + (current / total) * 35;
						callbacks.onProgress?.(progressPercent, status);
					},
				);

			callbacks.onProgress?.(
				40,
				`Fetched ${data.transactions.length} transactions`,
			);

			// Step 2: Analyze Meteora transactions
			callbacks.onProgress?.(45, "Analyzing Meteora transactions...");

			// Store the Meteora instructions
			const meteoraInstructions =
				await this.transactionService.analyzeMeteoraBatches(
					METEORA_PROGRAM_ID,
					(transactions, instructions) => {
						// Optional processing here if needed
					},
					(status, current, total) => {
						// Map progress from 45% to 90%
						const progressPercent = 45 + (current / total) * 45;
						callbacks.onProgress?.(
							progressPercent,
							`Meteora analysis: ${status}`,
						);
					},
				);

			// Store Meteora data in the result
			data.meteora = meteoraInstructions;

			callbacks.onProgress?.(
				90,
				`Analyzed ${meteoraInstructions.length} Meteora instructions`,
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
