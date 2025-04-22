// Export types
export type {
	DataDownloaderConfig,
	TransactionData,
	MeteoraData,
	TokenPriceData,
	DownloadedData,
} from "./src/types";

// Export main classes
export { DataDownloader } from "./src/DataDownloader";
export { Config } from "./src/config";

// Export services (for advanced usage)
export { TransactionService } from "./src/services/TransactionService";
export { MeteoraService } from "./src/services/MeteoraService";
export { TokenPriceService } from "./src/services/TokenPriceService";

// Example usage
if (require.main === module) {
	console.log("Mercon - Solana Data Downloader Utility");
	console.log("----------------------------------------");
	console.log("To use this library, import it in your code:");
	console.log('import { DataDownloader } from "mercon";');
	console.log("\nOr use it with environment variables:");
	console.log("WALLET_ADDRESS=... RPC_URL=... npx mercon download");
}
