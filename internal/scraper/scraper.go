package scraper

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/wnt/mercon/internal/models"
	"github.com/wnt/mercon/internal/solana"
	"gorm.io/gorm"
)

// Scraper represents the data scraping service
type Scraper struct {
	db            *gorm.DB
	solanaClient  *solana.Client
	txParser      *solana.TransactionParser
	maxConcurrent int
	requestTimeout time.Duration
}

// NewScraper creates a new instance of the scraper
func NewScraper(db *gorm.DB) (*Scraper, error) {
	// Create solana client
	solanaClient, err := solana.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	// Get max concurrent requests from env or use default
	maxConcurrentStr := os.Getenv("MAX_CONCURRENT_REQUESTS")
	maxConcurrent := 5 // default
	if maxConcurrentStr != "" {
		if val, err := strconv.Atoi(maxConcurrentStr); err == nil && val > 0 {
			maxConcurrent = val
		}
	}

	// Get request timeout from env or use default
	timeoutStr := os.Getenv("REQUEST_TIMEOUT")
	timeout := 30 * time.Second // default
	if timeoutStr != "" {
		if val, err := time.ParseDuration(timeoutStr); err == nil && val > 0 {
			timeout = val
		}
	}

	// Create transaction parser
	txParser := solana.NewTransactionParser(db, solanaClient)

	return &Scraper{
		db:            db,
		solanaClient:  solanaClient,
		txParser:      txParser,
		maxConcurrent: maxConcurrent,
		requestTimeout: timeout,
	}, nil
}

// Run executes the scraping operation
func (s *Scraper) Run() error {
	fmt.Println("Starting scraper...")

	// Get wallet address from env
	walletAddress := os.Getenv("WALLET_ADDRESS")
	if walletAddress == "" {
		return fmt.Errorf("WALLET_ADDRESS environment variable is not set")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	// Get or create wallet record
	var wallet models.Wallet
	result := s.db.Where("address = ?", walletAddress).FirstOrCreate(&wallet, models.Wallet{
		Address: walletAddress,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to get or create wallet: %w", result.Error)
	}

	// Get recent transactions (limit to 10 for testing)
	txSigns, err := s.solanaClient.GetTransactionSigns(ctx, walletAddress)
	if err != nil {
		return fmt.Errorf("failed to get transactions: %w", err)
	}


	fmt.Printf("Found %d transactions for wallet %s\n", len(txSigns), walletAddress)

	// Process transactions
	

	// Update wallet record with last scraped time
	wallet.LastScraped = time.Now()
	wallet.TransactionCount = len(txSigns)
	if len(txSigns) > 0 {
		wallet.LastScraped = time.Now()
	}
	
	if err := s.db.Save(&wallet).Error; err != nil {
		return fmt.Errorf("failed to update wallet record: %w", err)
	}

	fmt.Println("Scraping completed successfully")
	return nil
} 