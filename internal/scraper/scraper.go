package scraper

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/wnt/mercon/internal/models"
	"github.com/wnt/mercon/internal/services"
	"github.com/wnt/mercon/internal/solana"
	"gorm.io/gorm"
)

// Scraper represents the data scraping service
type Scraper struct {
	db             *gorm.DB
	solanaClient   *solana.Client
	txParser       *solana.TransactionParser
	dataEnricher   *services.MeteoraDataEnricher
	maxConcurrent  int
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
	txParser := solana.NewTransactionParser(solanaClient)

	// Create data enricher
	dataEnricher := services.NewMeteoraDataEnricher(db)

	return &Scraper{
		db:             db,
		solanaClient:   solanaClient,
		txParser:       txParser,
		dataEnricher:   dataEnricher,
		maxConcurrent:  maxConcurrent,
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

	var txs []*models.Transaction
	var err error

	// Use the new method to get transactions filtered by program ID
	fmt.Printf("Fetching transactions for wallet %s\n", walletAddress)
	txs, err = s.solanaClient.GetAndParseTransactions(ctx, walletAddress, solana.Filters{})
	if err != nil {
		return fmt.Errorf("failed to get and filter transactions: %w", err)
	}
	fmt.Printf("Found %d transactions for wallet %s\n", len(txs), walletAddress)

	// Save the transactions to the database
	if len(txs) > 0 {
		if err := solana.SaveTransactions(s.db, wallet.ID, txs); err != nil {
			return fmt.Errorf("failed to save transactions: %w", err)
		}
		fmt.Printf("Saved %d transactions to the database\n", len(txs))
	}

	// Update wallet record with last scraped time
	wallet.LastScraped = time.Now()
	wallet.TransactionCount = len(txs)

	if err := s.db.Save(&wallet).Error; err != nil {
		return fmt.Errorf("failed to update wallet record: %w", err)
	}

	// Perform initial data enrichment
	fmt.Println("Enriching data with USD values...")
	s.enrichData()

	fmt.Println("Scraping completed successfully")
	return nil
}

// enrichData performs data enrichment for all entities
func (s *Scraper) enrichData() {
	// Enrich pairs with USD values
	if err := s.dataEnricher.EnrichPairs(); err != nil {
		fmt.Printf("Warning: Failed to enrich pairs with USD values: %v\n", err)
	} else {
		fmt.Println("Successfully enriched pairs with USD values")
	}

	// Enrich positions with performance metrics
	if err := s.dataEnricher.EnrichPositions(); err != nil {
		fmt.Printf("Warning: Failed to enrich positions with USD values: %v\n", err)
	} else {
		fmt.Println("Successfully enriched positions with USD values")
	}
}
