import { DataDownloader, TransactionService } from "../index";
import type { DataDownloaderConfig } from "../index";

// Example 1: Using the DataDownloader class
async function runDownloaderExample() {
	console.log("Mercon - Data Downloader Example");
	console.log("---------------------------------------");

	// Replace these values with your own
	const config: DataDownloaderConfig = {
		walletAddress: "BpYUs2g6QyyMdagmgEUzpbvCH8SBHopjejhkAe2Kcbmq", // Replace with a valid Solana wallet address
		rpcUrl: "https://grateful-jerrie-fast-mainnet.helius-rpc.com", // Replace with your RPC URL
		callbacks: {
			onProgress: (progress, message) => {
				// Update progress bar
				const barLength = 30;
				const filledLength = Math.round(barLength * (progress / 100));
				const bar =
					"█".repeat(filledLength) + "░".repeat(barLength - filledLength);

				// Clear line and print progress
				process.stdout.write(`\r[${bar}] ${progress.toFixed(1)}% - ${message}`);
			},
			onDone: (data) => {
				console.log("\n\nDownload completed!");
				console.log(`Transactions: ${data.transactions?.length || 0}`);
				console.log(`Meteora Data: ${data.meteora?.length || 0}`);
				console.log(`Token Prices: ${data.tokenPrices?.length || 0}`);

				// Print a sample of the downloaded data
				if (data.transactions && data.transactions.length > 0) {
					console.log("\nTransaction Sample:");
					console.log(data.transactions[0]);
				}

				if (data.meteora && data.meteora.length > 0) {
					console.log("\nMeteora Data Sample:");
					console.log(data.meteora[0]);
				}

				if (data.tokenPrices && data.tokenPrices.length > 0) {
					console.log("\nToken Price Sample:");
					console.log(data.tokenPrices[0]);
				}
			},
			onError: (error) => {
				console.error("\nError during download:", error.message);
			},
		},
	};

	// Create the downloader
	const downloader = new DataDownloader(config);

	try {
		// Start download, limiting to 50 transactions for the example
		await downloader.download(50);
	} catch (error) {
		console.error("Example failed:", error);
	}
}

// Example 2: Using TransactionService directly with the new batch methods
async function runTransactionServiceExample() {
	console.log("\n\nMercon - Direct TransactionService Example");
	console.log("---------------------------------------");

	// Replace these values with your own
	const walletAddress = "BpYUs2g6QyyMdagmgEUzpbvCH8SBHopjejhkAe2Kcbmq";
	const rpcUrl = "https://grateful-jerrie-fast-mainnet.helius-rpc.com";

	// Create the transaction service
	const transactionService = new TransactionService(rpcUrl, walletAddress);

	try {
		// Create a progress bar rendering function
		const renderProgress = (status: string, current: number, total: number) => {
			const progress = total > 0 ? (current / total) * 100 : 0;
			const barLength = 30;
			const filledLength = Math.round(barLength * (progress / 100));
			const bar =
				"█".repeat(filledLength) + "░".repeat(barLength - filledLength);

			// Clear line and print progress
			process.stdout.write(
				`\r[${bar}] ${progress.toFixed(1)}% - ${status} (${current}/${total || "?"})`,
			);
		};

		console.log("Fetching transactions in batches with progress updates:");
		const transactions = await transactionService.getTransactionsInBatches(
			300, // Batch size
			renderProgress, // Progress callback
		);

		console.log("\n\nFetched transactions:", transactions.length);
		if (transactions.length > 0) {
			console.log("First transaction sample:");
			console.log(transactions[0]);
		}

		// Example with Meteora analysis (commented out for now)
		/*
		console.log("\nAnalyzing Meteora transactions:");
		const meteoraTransactions = await transactionService.analyzeMeteoraBatches(
			"your_meteora_program_id_here",
			(batchTransactions) => {
				// Process each batch as it arrives
				console.log(`\nProcessing batch of ${batchTransactions.length} transactions`);
			},
			renderProgress
		);
		*/
	} catch (error) {
		console.error("Transaction service example failed:", error);
	}
}

// Run the examples
async function runAllExamples() {
	await runDownloaderExample();
	await runTransactionServiceExample();
}

runAllExamples().catch(console.error);
