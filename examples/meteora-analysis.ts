import { TransactionService } from "../index";
import type { TransactionData } from "../index";

/**
 * Example showing how to use the TransactionService to analyze Meteora transactions
 * with concurrent processing as they are downloaded.
 */
async function runMeteoraAnalysisExample() {
	console.log("Mercon - Meteora Analysis Example");
	console.log("---------------------------------------");

	// Replace these values with your own
	const walletAddress = "BpYUs2g6QyyMdagmgEUzpbvCH8SBHopjejhkAe2Kcbmq";
	const rpcUrl = "https://grateful-jerrie-fast-mainnet.helius-rpc.com";

	// Meteora program ID - replace with the actual Meteora program ID
	const METEORA_PROGRAM_ID = "M2mx93ekt1fmXSVkTrUL9xVFHkmME8HTUi5Cyc5aF7K";

	// Create the transaction service
	const transactionService = new TransactionService(rpcUrl, walletAddress);

	// Counters for different Meteora operations
	const stats = {
		createPositions: 0,
		addLiquidity: 0,
		removeLiquidity: 0,
		closePositions: 0,
		total: 0,
		processed: 0,
	};

	try {
		// Create a progress bar rendering function
		const renderProgress = (status: string, current: number, total: number) => {
			const progress = total > 0 ? (current / total) * 100 : 0;
			const barLength = 30;
			const filledLength = Math.round(barLength * (progress / 100));
			const bar =
				"█".repeat(filledLength) + "░".repeat(barLength - filledLength);

			// Current stats
			const statsLine = `Meteora: Create=${stats.createPositions}, Add=${stats.addLiquidity}, Remove=${stats.removeLiquidity}, Close=${stats.closePositions}`;

			// Clear line and print progress
			process.stdout.write(
				`\r[${bar}] ${progress.toFixed(1)}% - ${status} (${current}/${total || "?"}) | ${statsLine}`,
			);
		};

		// Function to analyze transaction batches as they arrive
		const analyzeTransactionBatch = (transactions: TransactionData[]) => {
			stats.processed += transactions.length;

			// This is where you would implement your actual Meteora transaction detection logic
			// For this example, we're just simulating finding different transaction types

			// Simulate finding Meteora transactions (in a real scenario, you would check transaction data)
			for (const tx of transactions) {
				// In a real implementation, you would check if this transaction involves the Meteora program
				// and identify the specific operation type based on the instruction data

				// Simulate randomly finding different Meteora operations
				const random = Math.random();

				if (random < 0.05) {
					// 5% chance to be a create position
					stats.createPositions++;
					stats.total++;
				} else if (random < 0.15) {
					// 10% chance to be add liquidity
					stats.addLiquidity++;
					stats.total++;
				} else if (random < 0.25) {
					// 10% chance to be remove liquidity
					stats.removeLiquidity++;
					stats.total++;
				} else if (random < 0.3) {
					// 5% chance to be close position
					stats.closePositions++;
					stats.total++;
				}
			}
		};

		console.log("Analyzing Meteora transactions as they are downloaded:");
		console.log(
			"(This will process all transactions and analyze them concurrently)",
		);

		// Use the analyzeMeteoraBatches method to process transactions as they arrive
		const meteoraTransactions = await transactionService.analyzeMeteoraBatches(
			METEORA_PROGRAM_ID,
			analyzeTransactionBatch,
			renderProgress,
		);

		console.log("\n\nAnalysis complete!");
		console.log("---------------------------------------");
		console.log("Total transactions processed:", stats.processed);
		console.log("Total Meteora transactions found:", stats.total);
		console.log("Create positions:", stats.createPositions);
		console.log("Add liquidity:", stats.addLiquidity);
		console.log("Remove liquidity:", stats.removeLiquidity);
		console.log("Close positions:", stats.closePositions);
	} catch (error) {
		console.error("\nMeteora analysis failed:", error);
	}
}

// Run the example
runMeteoraAnalysisExample().catch(console.error);
