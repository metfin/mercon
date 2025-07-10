package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/joho/godotenv"
	"github.com/metfin/core/parsers/damm"
	"github.com/metfin/core/parsers/dlmm"
	externalCache "github.com/metfin/external/cache"
	externalConfig "github.com/metfin/external/config"
	externalRPC "github.com/metfin/external/rpc"
	externalServices "github.com/metfin/external/service"
)

func main() {
	// Create timestamped log file
	timestamp := time.Now().Format("20060102-150405")
	logFileName := fmt.Sprintf("log-%s.log", timestamp)

	logFile, err := os.Create(logFileName)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	// Create a multi-writer to write to both stdout and log file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Redirect log output to our multi-writer
	log.SetOutput(multiWriter)

	// Create a custom printf function that writes to both outputs
	printf := func(format string, args ...interface{}) {
		fmt.Fprintf(multiWriter, format, args...)
	}

	println := func(args ...interface{}) {
		fmt.Fprintln(multiWriter, args...)
	}

	// Parse command line arguments
	var walletAddress string
	var limit int
	flag.StringVar(&walletAddress, "wallet", "", "Wallet address to scan (required)")
	flag.IntVar(&limit, "limit", 3000, "Maximum number of transactions to process")
	flag.Parse()

	if walletAddress == "" {
		println("Usage: go run main.go -wallet <wallet_address> [-limit <number>]")
		println("Example: go run main.go -wallet 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM -limit 50")
		os.Exit(1)
	}

	printf("🔍 Scanning wallet: %s\n", walletAddress)
	printf("📊 Transaction limit: %d\n", limit)
	printf("📄 Log file: %s\n", logFileName)
	println(strings.Repeat("=", 80))

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Load external service configuration
	cfg := externalConfig.LoadConfig()

	// Initialize cache and RPC pool
	cache := externalCache.NewMemoryCache(time.Duration(cfg.CacheTTLSeconds) * time.Second)
	// we dont need websockets in mercon
	rpcPool := externalRPC.NewRPCPool(cfg.SolanaRPCURLs, []string{})
	defer rpcPool.CloseAll()

	// Create external service
	svc := externalServices.NewExternalService(
		rpcPool,
		cache,
		cfg.MeteoraDAMMBaseURL,
		cfg.MeteoraDLMMBaseURL,
		cfg.DexScreenerBaseURL,
		cfg.RugCheckBaseURL,
		cfg.JupBaseURL,
		cfg.CoingeckoBaseURL,
		cfg.BirdeyeBaseURL,
		cfg.PumpFunBaseURL,
	)

	ctx := context.Background()

	// Get transaction signatures for the wallet
	println("🔄 Fetching transaction signatures...")
	signatures, err := getWalletTransactionSignatures(ctx, svc, walletAddress, limit)
	if err != nil {
		log.Fatalf("❌ Failed to get wallet transactions: %v", err)
	}

	if len(signatures) == 0 {
		println("📭 No transactions found for this wallet")
		return
	}

	printf("✅ Found %d transaction signatures\n", len(signatures))

	// Fetch transaction details in batches
	println("🔄 Fetching transaction details...")
	transactions, err := getTransactionsInBulk(ctx, svc, signatures)
	if err != nil {
		log.Fatalf("❌ Failed to get transaction details: %v", err)
	}

	printf("✅ Fetched %d transactions\n", len(transactions))
	println(strings.Repeat("=", 80))

	// Process and parse each transaction
	var totalParsed int
	var dammCount, dlmmCount int

	for i, txResult := range transactions {
		if txResult == nil || txResult.Transaction == nil {
			continue
		}

		printf("\n🔍 Transaction #%d\n", i+1)
		printf("📝 Signature: %s\n", signatures[i])

		if txResult.BlockTime != nil {
			blockTime := time.Unix(int64(*txResult.BlockTime), 0)
			printf("⏰ Block Time: %s\n", blockTime.Format("2006-01-02 15:04:05 UTC"))
		}

		if txResult.Slot != 0 {
			printf("🎯 Slot: %d\n", txResult.Slot)
		}

		// Convert transaction to a format we can parse
		tx, err := txResult.Transaction.GetTransaction()
		if err != nil {
			printf("❌ Failed to parse transaction: %v\n", err)
			continue
		}

		// Parse DAMM instructions
		dammInstructions, err := damm.ParseDAMMTransaction(tx.Message.Instructions, tx.Message.AccountKeys)
		if err != nil {
			printf("⚠️  Failed to parse DAMM instructions: %v\n", err)
		} else if len(dammInstructions) > 0 {
			printf("🟢 DAMM Instructions Found: %d\n", len(dammInstructions))
			dammCount += len(dammInstructions)
			for j, inst := range dammInstructions {
				printf("  %d. Type: %s\n", j+1, inst.Type)
				if inst.Parsed != nil {
					printf("     Parsed Data: %s\n", formatInstruction(inst.Parsed))
				}
			}
		}

		// Parse DLMM instructions
		dlmmInstructions, err := dlmm.ParseDLMMTransaction(tx.Message.Instructions, tx.Message.AccountKeys)
		if err != nil {
			printf("⚠️  Failed to parse DLMM instructions: %v\n", err)
		} else if len(dlmmInstructions) > 0 {
			printf("🔵 DLMM Instructions Found: %d\n", len(dlmmInstructions))
			dlmmCount += len(dlmmInstructions)
			for j, inst := range dlmmInstructions {
				printf("  %d. Type: %s\n", j+1, inst.Type)
				if inst.Parsed != nil {
					printf("     Parsed Data: %s\n", formatInstruction(inst.Parsed))
				}
			}
		}

		// Show if no relevant instructions found
		if len(dammInstructions) == 0 && len(dlmmInstructions) == 0 {
			printf("⚪ No Meteora instructions found\n")
		} else {
			totalParsed++
		}

		println(strings.Repeat("-", 40))
	}

	// Summary
	printf("\n📈 SCAN SUMMARY\n")
	println(strings.Repeat("=", 80))
	printf("💳 Wallet: %s\n", walletAddress)
	printf("🔢 Total Transactions Scanned: %d\n", len(transactions))
	printf("✅ Transactions with Meteora Instructions: %d\n", totalParsed)
	printf("🟢 Total DAMM Instructions: %d\n", dammCount)
	printf("🔵 Total DLMM Instructions: %d\n", dlmmCount)
	printf("⚡ Scan completed successfully!\n")
	printf("📄 Full log saved to: %s\n", logFileName)
}

// Helper function to get wallet transaction signatures
func getWalletTransactionSignatures(ctx context.Context, svc externalServices.ExternalService, walletAddress string, limit int) ([]string, error) {
	// Use type assertion to access the method
	extImpl, ok := svc.(interface {
		GetWalletTransactionSignatures(ctx context.Context, walletAddress string, limit int) ([]string, error)
	})
	if !ok {
		return nil, fmt.Errorf("external service does not support GetWalletTransactionSignatures")
	}

	return extImpl.GetWalletTransactionSignatures(ctx, walletAddress, limit)
}

// Helper function to get transactions in bulk
func getTransactionsInBulk(ctx context.Context, svc externalServices.ExternalService, signatures []string) ([]*rpc.GetTransactionResult, error) {
	// Use type assertion to access the method
	extImpl, ok := svc.(interface {
		GetTransactionsInBulk(ctx context.Context, txHashes []string) ([]*rpc.GetTransactionResult, error)
	})
	if !ok {
		return nil, fmt.Errorf("external service does not support GetTransactionsInBulk")
	}

	return extImpl.GetTransactionsInBulk(ctx, signatures)
}

// Helper function to format instruction data for display
func formatInstruction(parsed interface{}) string {
	if parsed == nil {
		return "N/A"
	}

	// Convert to JSON for pretty printing
	data, err := json.MarshalIndent(parsed, "     ", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", parsed)
	}

	return string(data)
}
