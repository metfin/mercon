import {
  DatabaseType,
  type MeteoraDownloaderConfig,
  type MeteoraDlmmDbTransactions,
  MeteoraDlmmDb,
} from "../index";

async function runDatabaseExample() {
  console.log("Mercon - Meteora Database Downloader Example");
  console.log("-------------------------------------------");

  // Replace with your values
  const walletAddress = "BpYUs2g6QyyMdagmgEUzpbvCH8SBHopjejhkAe2Kcbmq";
  const rpcUrl = "https://grateful-jerrie-fast-mainnet.helius-rpc.com";

  // Create the database
  console.log("Creating database...");
  const meteoraDb = await MeteoraDlmmDb.create({
    type: DatabaseType.SQLITE,
    path: "./meteora.db", // Save in the current directory
  });

  try {
    // Check if already downloaded
    const isComplete = await meteoraDb.isComplete(walletAddress);
    if (isComplete) {
      console.log("Account already fully downloaded. Getting statistics...");
      const transactions = await meteoraDb.getOwnerTransactions(walletAddress);
      console.log(
        `Account has ${transactions.length} transactions in the database.`
      );

      // Show some download stats
      const openCount = transactions.filter(
        (tx: MeteoraDlmmDbTransactions) => tx.is_opening_transaction === 1
      ).length;
      const closeCount = transactions.filter(
        (tx: MeteoraDlmmDbTransactions) => tx.is_closing_transaction === 1
      ).length;
      console.log(`- ${openCount} position openings`);
      console.log(`- ${closeCount} position closings`);
      console.log(
        `- ${transactions.length - openCount - closeCount} other transactions`
      );

      // Show a few sample transactions
      if (transactions.length > 0) {
        console.log("\nSample transactions:");
        transactions
          .slice(0, 3)
          .forEach((tx: MeteoraDlmmDbTransactions, i: number) => {
            console.log(`\nTransaction ${i + 1}:`);
            console.log(`  Signature: ${tx.signature}`);
            console.log(`  Position: ${tx.position_address}`);
            console.log(
              `  Type: ${tx.is_opening_transaction ? "Open" : tx.is_closing_transaction ? "Close" : "Other"}`
            );
            console.log(`  Tokens: ${tx.base_symbol}/${tx.quote_symbol}`);
          });
      }
    } else {
      // Configure downloader
      const downloaderConfig: MeteoraDownloaderConfig = {
        account: walletAddress,
        rpcUrl,
        callbacks: {
          onProgress: (status, current, total) => {
            const progress =
              total > 0 ? Math.round((current / total) * 100) : 0;
            process.stdout.write(
              `\r[${progress.toString().padStart(3)}%] ${status}`
            );
          },
          onDone: () => {
            console.log("\nDownload completed!");
          },
          onError: (error) => {
            console.error("\nError during download:", error.message);
          },
        },
      };

      // Create and start downloader
      console.log("Starting download...");
      const downloader = meteoraDb.download(downloaderConfig);

      // Start the download
      await downloader.start();

      // Monitor status (in a real app, you might want to poll this from another thread)
      setTimeout(() => {
        const status = downloader.getStatus();
        console.log("\nCurrent status:", status.status);
        console.log(
          `Progress: ${status.progress.percentage}% (${status.progress.current}/${status.progress.total})`
        );
      }, 5000);

      // Example of how to cancel (commented out)
      // setTimeout(() => {
      //   console.log("\nCancelling download...");
      //   downloader.cancel();
      // }, 10000);

      // Wait for download to complete
      const status = downloader.getStatus();
      console.log(`\nFinal status: ${status.status}`);

      // Show transaction stats
      const transactions = await meteoraDb.getOwnerTransactions(walletAddress);
      console.log(`\nDownloaded ${transactions.length} transactions.`);
    }
  } catch (error) {
    console.error("Error:", error);
  } finally {
    // Close the database
    await meteoraDb.close();
    console.log("\nDatabase closed.");
  }
}

// Run the example
runDatabaseExample().catch(console.error);
