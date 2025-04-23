import { PublicKey } from "@solana/web3.js";
import type { MeteoraDlmmInstruction } from "../services/MeteoraParser";

// Configuration type for the data downloader
export interface DataDownloaderConfig {
	walletAddress: string; // Solana public key as string
	rpcUrl: string; // RPC URL for Solana connection
	callbacks: {
		onDone?: (data: DownloadedData) => void;
		onProgress?: (progress: number, message: string) => void;
		onError?: (error: Error) => void;
	};
}

// Types for transaction data
export interface TransactionData {
	signature: string;
	timestamp: number;
	blockTime: number;
	slot: number;
	status: "success" | "failed";
	// Add more fields as needed
}

// Types for token price data
export interface TokenPriceData {
	symbol: string;
	price: number;
	timestamp: number;
	// Add more price-specific fields as needed
}

// Combined data output
export interface DownloadedData {
	transactions?: TransactionData[];
	meteora?: MeteoraDlmmInstruction[];
	tokenPrices?: TokenPriceData[];
}
