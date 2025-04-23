import { TransactionService } from "../index";
import type { TransactionData } from "../index";
import { METEORA_PROGRAM_ID } from "../src/services/MeteoraParser";
import type { MeteoraDlmmInstruction } from "../src/services/MeteoraParser";

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
        `\r[${bar}] ${progress.toFixed(1)}% - ${status} (${current}/${total || "?"}) | ${statsLine}`
      );
    };

    // Function to analyze transaction batches as they arrive
    const analyzeTransactionBatch = (
      transactions: TransactionData[],
      meteoraInstructions: MeteoraDlmmInstruction[]
    ) => {
      stats.processed += transactions.length;
      stats.total += meteoraInstructions.length;

      // Process the actual Meteora instructions
      for (const instruction of meteoraInstructions) {
        // Count by instruction type
        switch (instruction.instructionType) {
          case "open":
            stats.createPositions++;
            break;
          case "add":
            stats.addLiquidity++;
            break;
          case "remove":
            stats.removeLiquidity++;
            break;
          case "close":
            stats.closePositions++;
            break;
        }
      }
    };

    console.log("Analyzing Meteora transactions as they are downloaded:");
    console.log(
      "(This will process all transactions and analyze them concurrently)"
    );

    // Use the analyzeMeteoraBatches method to process transactions as they arrive
    const meteoraInstructions = await transactionService.analyzeMeteoraBatches(
      METEORA_PROGRAM_ID,
      analyzeTransactionBatch,
      renderProgress
    );

    console.log("\n\nAnalysis complete!");
    console.log("---------------------------------------");
    console.log("Total transactions processed:", stats.processed);
    console.log("Total Meteora instructions found:", stats.total);
    console.log("Create positions:", stats.createPositions);
    console.log("Add liquidity:", stats.addLiquidity);
    console.log("Remove liquidity:", stats.removeLiquidity);
    console.log("Close positions:", stats.closePositions);

    // Display some sample instructions if available
    if (meteoraInstructions.length > 0) {
      console.log("\nSample Meteora instruction:");
      console.log(meteoraInstructions[0]);
    }
  } catch (error) {
    console.error("\nMeteora analysis failed:", error);
  }
}

// Run the example
runMeteoraAnalysisExample().catch(console.error);
