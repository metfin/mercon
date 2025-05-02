import type { DownloadedData } from "./downloaded-data";

// Configuration type for the data downloader
export interface DataDownloaderConfig {
  walletAddress: string; // Solana public key as string
  rpcUrl: string; // RPC URL for Solana connection
  callbackInterval?: number;
  onProgress?: (
    progress: number,
    total: number,
    statusType?: string,
    currentItem?: number,
    totalItems?: number
  ) => void;
  onDone?: (data: DownloadedData) => void;
  onError?: (error: Error) => void;
}

// Transaction data structure
export interface TransactionData {
  signature: string;
  slot: number;
  blockTime: number;
  data: unknown; // Raw transaction data
  parsed?: unknown; // Parsed transaction data (optional)
}

// Meteora data structure
export interface MeteoraData {
  positions: string[];
  pairs: string[];
}

// Token price data structure
export interface TokenPriceData {
  price: number;
  timestamp: number;
}
