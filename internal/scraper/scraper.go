package scraper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/wnt/mercon/internal/models"
	"github.com/wnt/mercon/internal/services"
	"github.com/wnt/mercon/internal/solana"
	"gorm.io/gorm"
)

// Default configuration values
const (
	DefaultMaxConcurrent  = 5
	DefaultRequestTimeout = 30 * time.Second
)

// ErrMissingWalletAddress is returned when the wallet address is not provided
var ErrMissingWalletAddress = errors.New("wallet address is not set")

// Scraper represents the data scraping service
type Scraper struct {
	db             *gorm.DB
	solanaClient   *solana.Client
	txParser       *solana.TransactionParser
	dataEnricher   *services.MeteoraDataEnricher
	maxConcurrent  int
	requestTimeout time.Duration
}

// Config holds the configuration for the scraper
type Config struct {
	MaxConcurrent  int
	RequestTimeout time.Duration
}

// NewScraper creates a new instance of the scraper
func NewScraper(db *gorm.DB) (*Scraper, error) {
	if db == nil {
		return nil, errors.New("database connection is required")
	}

	// Create solana client
	solanaClient, err := solana.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	config := loadConfigFromEnv()

	// Create transaction parser
	txParser := solana.NewTransactionParser(solanaClient)

	// Create data enricher
	dataEnricher := services.NewMeteoraDataEnricher(db)

	return &Scraper{
		db:             db,
		solanaClient:   solanaClient,
		txParser:       txParser,
		dataEnricher:   dataEnricher,
		maxConcurrent:  config.MaxConcurrent,
		requestTimeout: config.RequestTimeout,
	}, nil
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() Config {
	config := Config{
		MaxConcurrent:  DefaultMaxConcurrent,
		RequestTimeout: DefaultRequestTimeout,
	}

	// Get max concurrent requests from env
	if maxConcurrentStr := os.Getenv("MAX_CONCURRENT_REQUESTS"); maxConcurrentStr != "" {
		if val, err := strconv.Atoi(maxConcurrentStr); err == nil && val > 0 {
			config.MaxConcurrent = val
		}
	}

	// Get request timeout from env
	if timeoutStr := os.Getenv("REQUEST_TIMEOUT"); timeoutStr != "" {
		if val, err := time.ParseDuration(timeoutStr); err == nil && val > 0 {
			config.RequestTimeout = val
		}
	}

	return config
}

// Run executes the scraping operation
func (s *Scraper) Run() error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	return s.RunWithContext(ctx)
}

// RunWithContext executes the scraping operation with the provided context
func (s *Scraper) RunWithContext(ctx context.Context) error {
	fmt.Println("Starting scraper...")

	// Get wallet address from env
	walletAddress := os.Getenv("WALLET_ADDRESS")
	if walletAddress == "" {
		return ErrMissingWalletAddress
	}

	// Check if context is already done
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with execution
	}

	wallet, err := s.getOrCreateWallet(ctx, walletAddress)
	if err != nil {
		return fmt.Errorf("wallet operation failed: %w", err)
	}

	// Check if context is done after wallet operation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with execution
	}

	txs, err := s.fetchTransactions(ctx, walletAddress)
	if err != nil {
		return err
	}

	// Check if context is done after fetching transactions
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with execution
	}

	if err := s.saveTransactionsAndUpdateWallet(wallet, txs); err != nil {
		return err
	}

	// Check if context is done after saving transactions
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue with execution
	}

	// Perform data enrichment
	fmt.Println("Enriching data with USD values...")
	s.enrichData()

	fmt.Println("Scraping completed successfully")
	return nil
}

// getOrCreateWallet retrieves or creates a wallet record
func (s *Scraper) getOrCreateWallet(ctx context.Context, address string) (*models.Wallet, error) {
	var wallet models.Wallet
	result := s.db.WithContext(ctx).Where("address = ?", address).FirstOrCreate(&wallet, models.Wallet{
		Address: address,
	})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get or create wallet: %w", result.Error)
	}
	return &wallet, nil
}

// fetchTransactions retrieves transactions for the specified wallet
func (s *Scraper) fetchTransactions(ctx context.Context, walletAddress string) ([]*models.Transaction, error) {
	fmt.Printf("Fetching transactions for wallet %s\n", walletAddress)
	txs, err := s.solanaClient.GetAndParseTransactions(ctx, walletAddress, solana.Filters{})
	if err != nil {
		return nil, fmt.Errorf("failed to get and filter transactions: %w", err)
	}
	fmt.Printf("Found %d transactions for wallet %s\n", len(txs), walletAddress)
	return txs, nil
}

// saveTransactionsAndUpdateWallet saves transactions and updates the wallet record
func (s *Scraper) saveTransactionsAndUpdateWallet(wallet *models.Wallet, txs []*models.Transaction) error {
	// Save the transactions to the database if there are any
	if len(txs) > 0 {
		if err := solana.SaveTransactions(s.db, wallet.ID, txs); err != nil {
			return fmt.Errorf("failed to save transactions: %w", err)
		}
		fmt.Printf("Saved %d transactions to the database\n", len(txs))
	}

	// Update wallet record with last scraped time
	wallet.LastScraped = time.Now()
	wallet.TransactionCount = len(txs)

	if err := s.db.Save(wallet).Error; err != nil {
		return fmt.Errorf("failed to update wallet record: %w", err)
	}

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
