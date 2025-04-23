import type { DatabaseInterface } from "./database/DatabaseInterface";
import {
	DatabaseType,
	type DatabaseConfig,
	createDatabase,
} from "./database/DatabaseFactory";
import type {
	MeteoraDlmmDownloader,
	MeteoraDownloaderConfig,
	DownloaderStatus,
} from "./MeteoraDlmmDownloader";
import type {
	MeteoraDlmmPairData,
	MeteoraPositionTransactions,
	MeteoraDlmmDbTransactions,
} from "./types/meteora";
import type { TokenMeta } from "./types/tokens";
import type { MeteoraDlmmInstruction } from "./services/MeteoraParser";
import { MeteoraDlmmDownloader as Downloader } from "./MeteoraDlmmDownloader";

/**
 * Main class for managing Meteora DLMM database operations
 */
export class MeteoraDlmmDb {
	private db: DatabaseInterface;
	private downloaders: Map<string, MeteoraDlmmDownloader> = new Map();

	/**
	 * Create a new Meteora DLMM Database manager
	 */
	private constructor(db: DatabaseInterface) {
		this.db = db;
	}

	/**
	 * Create a new database instance
	 */
	static async create(
		config: DatabaseConfig = { type: DatabaseType.SQLITE },
	): Promise<MeteoraDlmmDb> {
		const db = await createDatabase(config);
		return new MeteoraDlmmDb(db);
	}

	/**
	 * Close the database connection
	 */
	async close(): Promise<void> {
		// Cancel all active downloaders
		for (const [, downloader] of this.downloaders.entries()) {
			downloader.cancel();
		}

		this.downloaders.clear();
		await this.db.save();
		await this.db.close();
	}

	/**
	 * Create a downloader for a specific account
	 */
	download(config: MeteoraDownloaderConfig): MeteoraDlmmDownloader {
		// Check if there's already a downloader for this account
		const existingDownloader = this.downloaders.get(config.account);
		if (existingDownloader) {
			return existingDownloader;
		}

		// Setup callbacks to remove downloader when done
		const callbacks = config.callbacks || {};

		// Wrap the onDone callback to remove the downloader from the map
		const originalOnDone = callbacks.onDone;
		callbacks.onDone = () => {
			this.downloaders.delete(config.account);
			originalOnDone?.();
		};

		// Create new downloader
		const downloader = new Downloader(this.db, {
			...config,
			callbacks,
		});

		// Store in the map
		this.downloaders.set(config.account, downloader);

		return downloader;
	}

	/**
	 * Get status for all active downloaders
	 */
	getDownloaderStatuses(): Record<string, DownloaderStatus> {
		const statuses: Record<string, DownloaderStatus> = {};

		for (const [account, downloader] of this.downloaders.entries()) {
			statuses[account] = downloader.getStatus();
		}

		return statuses;
	}

	/**
	 * Cancel a specific downloader
	 */
	async cancelDownload(account: string): Promise<void> {
		const downloader = this.downloaders.get(account);
		if (downloader) {
			downloader.cancel();
			this.downloaders.delete(account);
		}

		await this.db.save();
	}

	/**
	 * Cancel all downloaders
	 */
	async cancelAllDownloads(): Promise<void> {
		for (const [, downloader] of this.downloaders.entries()) {
			downloader.cancel();
		}

		this.downloaders.clear();
		await this.db.save();
	}

	/**
	 * Check if an account is already complete in the database
	 */
	async isComplete(account: string): Promise<boolean> {
		return this.db.isComplete(account);
	}

	/**
	 * Get all transactions for a specific owner
	 */
	async getOwnerTransactions(
		ownerAddress: string,
	): Promise<MeteoraDlmmDbTransactions[]> {
		return this.db.getOwnerTransactions(ownerAddress);
	}

	/**
	 * Get all transactions in the database
	 */
	async getAllTransactions(): Promise<MeteoraDlmmDbTransactions[]> {
		return this.db.getAllTransactions();
	}

	/**
	 * Add a token to the database
	 */
	async addToken(token: TokenMeta): Promise<void> {
		await this.db.addToken(token);
	}

	/**
	 * Add a pair to the database
	 */
	async addPair(pair: MeteoraDlmmPairData): Promise<void> {
		await this.db.addPair(pair);
	}

	/**
	 * Get LB pair for a position
	 */
	async getLbPair(positionAddress: string): Promise<string | undefined> {
		return this.db.getLbPair(positionAddress);
	}

	/**
	 * Add instruction to the database
	 */
	async addInstruction(instruction: MeteoraDlmmInstruction): Promise<void> {
		await this.db.addInstruction(instruction);
		await this.db.addTransfers(instruction);
	}

	/**
	 * Add USD transaction data
	 */
	async addUsdTransactions(
		positionAddress: string,
		transactions: MeteoraPositionTransactions,
	): Promise<void> {
		await this.db.addUsdTransactions(positionAddress, transactions);
	}

	/**
	 * Save the database state
	 */
	async save(): Promise<void> {
		await this.db.save();
	}

	/**
	 * Wait for all pending operations to complete
	 */
	async waitForSave(): Promise<void> {
		await this.db.waitForSave();
	}
}
