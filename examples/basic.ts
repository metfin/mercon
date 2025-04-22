import { DataDownloader } from "../index";
import type { DataDownloaderConfig } from "../index";

// Create a simple example to demonstrate how to use the library
async function runExample() {
	console.log("Mercon - Data Downloader Example");
	console.log("---------------------------------------");

	// Replace these values with your own
	const config: DataDownloaderConfig = {
		walletAddress: "YOUR_WALLET_ADDRESS", // Replace with a valid Solana wallet address
		rpcUrl: "https://api.mainnet-beta.solana.com", // Replace with your RPC URL
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

// Run the example
runExample().catch(console.error);
