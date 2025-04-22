import {
	Connection,
	PublicKey,
	type ConfirmedSignatureInfo,
	type ParsedTransactionWithMeta,
} from "@solana/web3.js";
import type { TransactionData } from "../types";

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
		limit = 100,
		before?: string,
		until?: string,
	): Promise<ConfirmedSignatureInfo[]> {
		try {
			return await this.connection.getSignaturesForAddress(this.walletAddress, {
				limit,
				before,
				until,
			});
		} catch (error) {
			console.error("Error fetching transaction signatures:", error);
			throw new Error(
				`Failed to fetch transaction signatures: ${error instanceof Error ? error.message : String(error)}`,
			);
		}
	}

	/**
	 * Fetch transaction details for a given signature
	 * @param signature Transaction signature
	 */
	public async getTransaction(
		signature: string,
	): Promise<ParsedTransactionWithMeta | null> {
		try {
			return await this.connection.getParsedTransaction(signature, {
				maxSupportedTransactionVersion: 0,
			});
		} catch (error) {
			console.error(`Error fetching transaction ${signature}:`, error);
			throw new Error(
				`Failed to fetch transaction ${signature}: ${error instanceof Error ? error.message : String(error)}`,
			);
		}
	}

	/**
	 * Fetch transactions in batches
	 * @param batchSize Size of each batch
	 */
	public async getTransactionsInBatches(
		batchSize = 300,
	): Promise<TransactionData[]> {
		const transactions: TransactionData[] = [];
		let before: string | undefined;
		const keepFetching = true;
		const fetchedCount = 0;

		while (keepFetching) {
			const signatures = await this.getTransactionSignatures(batchSize, before);

			if (signatures.length === 0) {
				break;
			}

			const signatureStrings = signatures.map((sig) => sig.signature);
			const batchTransactions = await this.getTransactions(signatureStrings);

			transactions.push(...batchTransactions);

			// Update the before parameter for the next batch
			if (signatures.length > 0) {
				before = signatures[signatures.length - 1]?.signature;
			} else {
				break;
			}
		}

		return transactions;
	}

	/**
	 * Fetch multiple transactions and process them
	 * @param signatures List of transaction signatures
	 */
	public async getTransactions(
		signatures: string[],
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
	 * Fetch all transactions for the wallet with optional pagination
	 * @param maxTransactions Maximum number of transactions to fetch (0 for all)
	 * @param onProgress Optional progress callback
	 */
	public async fetchAllTransactions(
		maxTransactions = 0,
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
