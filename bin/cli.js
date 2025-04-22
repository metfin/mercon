#!/usr/bin/env node

import { DataDownloader } from "../index.js";

const command = process.argv[2];

function showHelp() {
	console.log("Mercon - Solana Data Downloader Utility");
	console.log("----------------------------------------");
	console.log("Usage:");
	console.log(
		"  mercon download [maxTransactions]  Download data for configured wallet",
	);
	console.log("  mercon --help                      Show this help message");
	console.log("\nEnvironment Variables:");
	console.log("  WALLET_ADDRESS                     Solana wallet address");
	console.log("  RPC_URL                            Solana RPC URL");
}

async function download() {
	const maxTransactions = Number.parseInt(process.argv[3], 10) || 1000;

	if (!process.env.WALLET_ADDRESS || !process.env.RPC_URL) {
		console.error(
			"Error: WALLET_ADDRESS and RPC_URL environment variables are required.",
		);
		console.error("Please set them before running the command:");
		console.error(
			"  WALLET_ADDRESS=your_wallet RPC_URL=your_rpc_url mercon download",
		);
		process.exit(1);
	}

	const downloader = DataDownloader.fromEnv();

	// Setup callbacks for CLI
	downloader.config.getCallbacks().onProgress = (progress, message) => {
		// Update progress bar
		const barLength = 30;
		const filledLength = Math.round(barLength * (progress / 100));
		const bar = "█".repeat(filledLength) + "░".repeat(barLength - filledLength);

		// Clear line and print progress
		process.stdout.write(`\r[${bar}] ${progress.toFixed(1)}% - ${message}`);
	};

	try {
		console.log(`Starting download for wallet ${process.env.WALLET_ADDRESS}`);
		console.log(`Max transactions: ${maxTransactions}`);

		const data = await downloader.download(maxTransactions);

		console.log("\n\nDownload completed!");
		console.log(`Transactions: ${data.transactions?.length || 0}`);
		console.log(`Meteora Data: ${data.meteora?.length || 0}`);
		console.log(`Token Prices: ${data.tokenPrices?.length || 0}`);
	} catch (error) {
		console.error("\nError during download:", error);
		process.exit(1);
	}
}

// Process commands
switch (command) {
	case "download":
		download().catch(console.error);
		break;

	case "--help":
	case "-h":
		showHelp();
		break;

	default:
		if (!command) {
			showHelp();
		} else {
			console.error(`Unknown command: ${command}`);
			console.error('Run "mercon --help" to see available commands');
			process.exit(1);
		}
		break;
}
