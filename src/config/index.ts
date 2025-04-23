import type { DataDownloaderConfig } from "../types";
import { PublicKey, Connection } from "@solana/web3.js";
import * as dotenv from "dotenv";
import type { DownloadedData } from "../types/downloaded-data";

// Load environment variables from .env file
dotenv.config();

export class Config {
  private config: DataDownloaderConfig;

  constructor(config: DataDownloaderConfig) {
    this.validateConfig(config);
    this.config = config;
  }

  private validateConfig(config: DataDownloaderConfig): void {
    if (!config.walletAddress) {
      throw new Error("Wallet address is required");
    }

    if (!config.rpcUrl) {
      throw new Error("RPC URL is required");
    }

    try {
      // Validate that the wallet address is a valid Solana public key
      new PublicKey(config.walletAddress);
    } catch (error) {
      throw new Error(
        `Invalid wallet address: ${error instanceof Error ? error.message : String(error)}`
      );
    }

    try {
      // Test RPC connection
      new Connection(config.rpcUrl);
    } catch (error) {
      throw new Error(
        `Invalid RPC URL: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }

  public getConfig(): DataDownloaderConfig {
    return this.config;
  }

  public getWalletAddress(): string {
    return this.config.walletAddress;
  }

  public getRpcUrl(): string {
    return this.config.rpcUrl;
  }

  public getCallbackInterval(): number | undefined {
    return this.config.callbackInterval;
  }

  public getOnProgress() {
    return this.config.onProgress;
  }

  public getOnDone() {
    return this.config.onDone;
  }

  public getOnError() {
    return this.config.onError;
  }

  public static createFromEnv(): Config {
    const config: DataDownloaderConfig = {
      walletAddress: process.env["WALLET_ADDRESS"] || "",
      rpcUrl: process.env["RPC_URL"] || "",
      onDone: (data: DownloadedData) => console.log("Download completed", data),
      onProgress: (progress: number, message: number) =>
        console.log(`Progress: ${progress}% - ${message}`),
      onError: (error: Error) => console.error("Error:", error),
    };

    return new Config(config);
  }
}
