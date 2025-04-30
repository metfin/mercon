#!/usr/bin/env node

import { DataDownloader, MeteoraDlmmDb, DatabaseType } from "../index.js";
import path from "node:path";
import os from "node:os";

const command = process.argv[2];
const subCommand = process.argv[3];

function showHelp() {
  console.log("Mercon - Solana Data Downloader Utility");
  console.log("----------------------------------------");
  console.log("Usage:");
  console.log(
    "  mercon download [maxTransactions]  Download data for configured wallet"
  );
  console.log(
    "  mercon db download [maxTx]         Download data to local database"
  );
  console.log(
    "  mercon db query [walletAddress]    Query transactions from database"
  );
  console.log(
    "  mercon db status [walletAddress]   Check download status for wallet"
  );
  console.log("  mercon --help                      Show this help message");
  console.log("\nEnvironment Variables:");
  console.log("  WALLET_ADDRESS                     Solana wallet address");
  console.log("  RPC_URL                            Solana RPC URL");
  console.log(
    "  DB_PATH                            Optional database path (default: ~/.mercon/meteora.db)"
  );
}

async function download() {
  const maxTransactions = Number.parseInt(process.argv[3], 10);

  if (!process.env.WALLET_ADDRESS || !process.env.RPC_URL) {
    console.error(
      "Error: WALLET_ADDRESS and RPC_URL environment variables are required."
    );
    console.error("Please set them before running the command:");
    console.error(
      "  WALLET_ADDRESS=your_wallet RPC_URL=your_rpc_url mercon download"
    );
    process.exit(1);
  }

  // Set up status line management
  let lastStatus = "";

  // Create the downloader with custom callbacks
  const customConfig = {
    walletAddress: process.env.WALLET_ADDRESS,
    rpcUrl: process.env.RPC_URL,
    // Add throttling config
    throttlingConfig: {
      requestsPerBatch: 30, // Reduce batch size
      delayBetweenBatches: 2000, // 2 seconds between batches
      delayAfterRateLimit: 5000, // 5 seconds after hitting rate limit
    },
    onProgress: (progress, total, statusType, currentItem, totalItems) => {
      // Clear previous status line if needed
      if (lastStatus) {
        process.stdout.clearLine(0);
        process.stdout.cursorTo(0);
      }

      // Build a nice progress indicator
      const barLength = 25;
      const filledLength = Math.round(barLength * (progress / 100));
      const bar =
        "█".repeat(filledLength) + "░".repeat(barLength - filledLength);

      // Create status message based on the operation type
      let statusMessage = "";

      switch (statusType) {
        case "FETCHING_SIGNATURES":
          statusMessage = `Fetching Transaction Signatures [${currentItem} found]`;
          break;
        case "PROCESSING_TRANSACTIONS":
          statusMessage = `Processing Transactions [Batch ${currentItem}/${totalItems}]`;
          break;
        case "ANALYZING_METEORA":
          statusMessage = `Analyzing Meteora Data [${Math.round(progress)}%]`;
          break;
        case "UPDATING_PRICES":
          statusMessage = `Updating Token Prices [${Math.round(progress)}%]`;
          break;
        case "RATE_LIMITED":
          statusMessage = `Rate limited, waiting ${currentItem}ms [${Math.round(progress)}%]`;
          break;
        default:
          statusMessage = `${statusType} [${Math.round(progress)}%]`;
      }

      // Build full status line
      const statusLine = `[${bar}] ${statusMessage}`;
      process.stdout.write(statusLine);
      lastStatus = statusLine;
    },
  };

  const downloader = new DataDownloader(customConfig);

  try {
    console.log("\nSolana Data Downloader");
    console.log(`Wallet: ${process.env.WALLET_ADDRESS}`);
    console.log(`Max transactions: ${maxTransactions}`);
    console.log(`RPC URL: ${process.env.RPC_URL}`);
    console.log("\nStarting download process...\n");

    const data = await downloader.download(maxTransactions);

    // Clear the last status line
    if (lastStatus) {
      process.stdout.clearLine(0);
      process.stdout.cursorTo(0);
    }

    console.log("\n✅ Download completed successfully!\n");
    console.log("📊 Summary:");
    console.log(`   Transactions: ${data.transactions}`);
    console.log(`   Meteora Positions: ${data.positions?.length || 0}`);
    console.log(`   Meteora Pairs: ${data.pairs?.length || 0}`);
    console.log(
      `   Time taken: ${((data.endTime.getTime() - data.startTime.getTime()) / 1000).toFixed(1)}s\n`
    );
  } catch (error) {
    // Clear the last status line
    if (lastStatus) {
      process.stdout.clearLine(0);
      process.stdout.cursorTo(0);
    }
    console.error("\n❌ Error during download:", error);
    process.exit(1);
  }
}

async function dbDownload() {
  const maxTransactions = Number.parseInt(process.argv[4], 10);

  if (!process.env.WALLET_ADDRESS || !process.env.RPC_URL) {
    console.error(
      "Error: WALLET_ADDRESS and RPC_URL environment variables are required."
    );
    console.error("Please set them before running the command:");
    console.error(
      "  WALLET_ADDRESS=your_wallet RPC_URL=your_rpc_url mercon db download"
    );
    process.exit(1);
  }

  // Create database
  const dbPath =
    process.env.DB_PATH || path.join(os.homedir(), ".mercon", "meteora.db");
  console.log("Creating database...");

  const meteoraDb = await MeteoraDlmmDb.create({
    type: DatabaseType.SQLITE,
    path: dbPath,
  });

  try {
    const walletAddress = process.env.WALLET_ADDRESS;
    const rpcUrl = process.env.RPC_URL;

    // Check if already downloaded
    const isComplete = await meteoraDb.isComplete(walletAddress);

    if (isComplete) {
      console.log("Account already fully downloaded. Getting statistics...");
      const transactions = await meteoraDb.getOwnerTransactions(walletAddress);
      console.log(
        `Account has ${transactions.length} transactions in the database.`
      );

      // Show summary and exit
      await showTransactionSummary(meteoraDb, walletAddress);
    } else {
      // Configure downloader with progress tracking and rate limit handling
      const downloaderConfig = {
        account: walletAddress,
        rpcUrl,
        maxTransactions: maxTransactions ? maxTransactions : undefined,
        batchSize: 100, // Reduce batch size
        callbacks: {
          onProgress: (status, current, total) => {
            const progress =
              total > 0 ? Math.round((current / total) * 100) : 0;
            process.stdout.clearLine?.(0);
            process.stdout.cursorTo?.(0);
            process.stdout.write(
              `[${progress.toString().padStart(3)}%] ${status}`
            );
          },
          onDone: () => {
            console.log("\n✅ Download completed successfully!");
          },
          onError: (error) => {
            console.error("\n❌ Error during download:", error.message);
          },
        },
      };

      // Start download
      console.log("\nStarting database download...");
      console.log(`Wallet: ${walletAddress}`);
      console.log(`Max transactions: ${maxTransactions}`);
      console.log(`Database: ${dbPath}`);
      console.log(`RPC URL: ${rpcUrl.substring(0, 20)}...`); // Truncate for privacy
      console.log(
        "\nNOTE: Using smaller batch sizes and delays to avoid rate limits"
      );
      console.log("");

      const downloader = meteoraDb.download(downloaderConfig);
      await downloader.start();

      // Show summary
      await showTransactionSummary(meteoraDb, walletAddress);
    }
  } catch (error) {
    console.error("\n❌ Error:", error);
    process.exit(1);
  } finally {
    // Close the database
    await meteoraDb.close();
    console.log("Database connection closed.");
  }
}

async function dbQuery() {
  const walletAddress = process.argv[4] || process.env.WALLET_ADDRESS;

  if (!walletAddress) {
    console.error("Error: Wallet address is required.");
    console.error(
      "Please provide it as an argument or set WALLET_ADDRESS environment variable:"
    );
    console.error("  mercon db query <wallet_address>");
    process.exit(1);
  }

  // Create database
  const dbPath =
    process.env.DB_PATH || path.join(os.homedir(), ".mercon", "meteora.db");

  const meteoraDb = await MeteoraDlmmDb.create({
    type: DatabaseType.SQLITE,
    path: dbPath,
  });

  try {
    // Show data for wallet
    await showTransactionSummary(meteoraDb, walletAddress);

    // Show sample transactions
    const transactions = await meteoraDb.getOwnerTransactions(walletAddress);
    if (transactions.length > 0) {
      console.log("\nSample transactions:");
      transactions.slice(0, 5).forEach((tx, i) => {
        console.log(`\nTransaction ${i + 1}:`);
        console.log(`  Signature: ${tx.signature}`);
        console.log(`  Position: ${tx.position_address}`);
        console.log(
          `  Type: ${tx.is_opening_transaction ? "Open" : tx.is_closing_transaction ? "Close" : "Other"}`
        );
        console.log(
          `  Tokens: ${tx.base_symbol || "Unknown"}/${tx.quote_symbol || "Unknown"}`
        );
      });
    }
  } catch (error) {
    console.error("❌ Error:", error);
    process.exit(1);
  } finally {
    await meteoraDb.close();
    console.log("\nDatabase connection closed.");
  }
}

async function dbStatus() {
  const walletAddress = process.argv[4] || process.env.WALLET_ADDRESS;

  if (!walletAddress) {
    console.error("Error: Wallet address is required.");
    console.error(
      "Please provide it as an argument or set WALLET_ADDRESS environment variable:"
    );
    console.error("  mercon db status <wallet_address>");
    process.exit(1);
  }

  // Create database
  const dbPath =
    process.env.DB_PATH || path.join(os.homedir(), ".mercon", "meteora.db");

  const meteoraDb = await MeteoraDlmmDb.create({
    type: DatabaseType.SQLITE,
    path: dbPath,
  });

  try {
    // Check status
    const isComplete = await meteoraDb.isComplete(walletAddress);
    console.log(`\nWallet: ${walletAddress}`);
    console.log(`Database: ${dbPath}`);
    console.log(
      `Download status: ${isComplete ? "✅ Complete" : "⏳ Incomplete or not started"}`
    );

    // Show more details if available
    const transactions = await meteoraDb.getOwnerTransactions(walletAddress);
    if (transactions.length > 0) {
      console.log(`\nTransactions in database: ${transactions.length}`);
      console.log(
        `Last transaction time: ${new Date(transactions[0].timestamp).toLocaleString()}`
      );
    } else {
      console.log("\nNo transactions found in database for this wallet.");
    }

    // Show active downloaders
    const statuses = meteoraDb.getDownloaderStatuses();
    if (Object.keys(statuses).length > 0) {
      console.log("\nActive downloaders:");
      for (const [account, status] of Object.entries(statuses)) {
        console.log(
          `  ${account}: ${status.status} (${status.progress.percentage}%)`
        );
      }
    }
  } catch (error) {
    console.error("❌ Error:", error);
    process.exit(1);
  } finally {
    await meteoraDb.close();
    console.log("\nDatabase connection closed.");
  }
}

async function showTransactionSummary(meteoraDb, walletAddress) {
  const transactions = await meteoraDb.getOwnerTransactions(walletAddress);

  // Calculate statistics
  const openCount = transactions.filter(
    (tx) => tx.is_opening_transaction === 1
  ).length;
  const closeCount = transactions.filter(
    (tx) => tx.is_closing_transaction === 1
  ).length;

  console.log(`\n📊 Transaction Summary for ${walletAddress}:`);
  console.log(`  Total Transactions: ${transactions.length}`);
  console.log(`  Position Openings: ${openCount}`);
  console.log(`  Position Closings: ${closeCount}`);
  console.log(
    `  Other Transactions: ${transactions.length - openCount - closeCount}`
  );

  return transactions;
}

// Process commands
switch (command) {
  case "download":
    download().catch(console.error);
    break;

  case "db":
    switch (subCommand) {
      case "download":
        dbDownload().catch(console.error);
        break;

      case "query":
        dbQuery().catch(console.error);
        break;

      case "status":
        dbStatus().catch(console.error);
        break;

      default:
        console.error(`Unknown db subcommand: ${subCommand}`);
        console.error('Run "mercon --help" to see available commands');
        process.exit(1);
    }
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
