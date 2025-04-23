// Export types
export type {
  DataDownloaderConfig,
  TransactionData,
  MeteoraData,
  TokenPriceData,
} from "./src/types";
export type { DownloadedData } from "./src/types/downloaded-data";

// Export main classes
export { DataDownloader } from "./src/DataDownloader";
export { Config } from "./src/config";

// Export services (for advanced usage)
export { TransactionService } from "./src/services/TransactionService";
export { MeteoraService } from "./src/services/MeteoraService";
export { TokenPriceService } from "./src/services/TokenPriceService";

// Main exports
export { MeteoraDlmmDb } from "./src/MeteoraDlmmDb";
export { MeteoraDlmmDownloader } from "./src/MeteoraDlmmDownloader";

// Database exports
export {
  DatabaseType,
  type DatabaseConfig,
} from "./src/database/DatabaseFactory";

// Type exports
export type {
  MeteoraDownloaderConfig,
  DownloaderStatus,
  DownloaderCallbacks,
} from "./src/MeteoraDlmmDownloader";

export type {
  MeteoraDlmmPairData,
  MeteoraPositionTransactions,
  MeteoraDlmmDbTransactions,
} from "./src/types/meteora";

export type { TokenMeta } from "./src/types/tokens";

// Service exports
export {
  METEORA_PROGRAM_ID,
  type MeteoraDlmmInstruction,
} from "./src/services/MeteoraParser";

// Example usage
if (require.main === module) {
  console.log("Mercon - Solana Data Downloader Utility");
  console.log("----------------------------------------");
  console.log("To use this library, import it in your code:");
  console.log('import { DataDownloader } from "mercon";');
  console.log("\nOr use it with environment variables:");
  console.log("WALLET_ADDRESS=... RPC_URL=... npx mercon download");
}
