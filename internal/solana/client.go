package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wnt/mercon/internal/constants"
	"github.com/wnt/mercon/internal/models"
	"github.com/wnt/mercon/internal/utils"
	"gorm.io/gorm"
)

// Default configuration values
const (
	DefaultTimeout = 30 * time.Second
)

// Error types for better error handling
var (
	ErrMissingRPCURL    = errors.New("RPC_URL environment variable is not set")
	ErrMissingAPIKey    = errors.New("HELIUS_API_KEY environment variable is not set")
	ErrEmptyTransaction = errors.New("no transactions to save")
)

// Client represents a connection to the Solana blockchain
type Client struct {
	rpcClient  *rpc.Client
	endpoint   string
	httpClient *utils.HTTPClient
}

// ClientConfig holds the configuration for the Solana client
type ClientConfig struct {
	RPCURL  string
	APIKey  string
	Timeout time.Duration
	BaseURL string
}

// Filters represents optional filters for transaction queries
type Filters struct {
	After  time.Time
	LastTx string
}

// NewClient creates a new Solana client
func NewClient() (*Client, error) {
	config, err := loadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	rpcClient := rpc.New(config.RPCURL)

	// Check connection by getting the latest block height
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	_, err = rpcClient.GetBlockHeight(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Solana RPC: %w", err)
	}

	// Create HTTP client for API requests
	httpClient := utils.NewHTTPClient(
		utils.WithTimeout(config.Timeout),
		utils.WithBaseURL(config.BaseURL),
	)

	return &Client{
		rpcClient:  rpcClient,
		endpoint:   config.RPCURL,
		httpClient: httpClient,
	}, nil
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() (ClientConfig, error) {
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		return ClientConfig{}, ErrMissingRPCURL
	}

	config := ClientConfig{
		RPCURL:  rpcURL,
		Timeout: DefaultTimeout,
		BaseURL: constants.HeliusBaseURL,
	}

	// Parse timeout if set
	if timeoutStr := os.Getenv("RPC_TIMEOUT"); timeoutStr != "" {
		if val, err := time.ParseDuration(timeoutStr); err == nil && val > 0 {
			config.Timeout = val
		}
	}

	return config, nil
}

// GetTransactions retrieves transactions for the specified wallet address
func (c *Client) GetTransactions(ctx context.Context, address string, filters Filters) ([]Transaction, error) {
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	// Build query parameters
	queryParams := map[string]string{
		"api-key": apiKey,
	}

	// Add optional filter parameters
	if !filters.After.IsZero() {
		queryParams["until"] = strconv.FormatInt(filters.After.Unix(), 10)
	}
	if filters.LastTx != "" {
		queryParams["before"] = filters.LastTx
	}

	// Make the request using the HTTP helper
	path := fmt.Sprintf("/addresses/%s/transactions", address)
	resp, err := c.httpClient.Get(path, queryParams, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Decode the response
	var transactions []Transaction
	if err := json.Unmarshal(resp.Body, &transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}

	return transactions, nil
}

// GetAndParseTransactions retrieves and parses transactions for the specified wallet address
func (c *Client) GetAndParseTransactions(ctx context.Context, address string, filters Filters) ([]*models.Transaction, error) {
	transactions, err := c.GetTransactions(ctx, address, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	if len(transactions) == 0 {
		return []*models.Transaction{}, nil
	}

	txParser := NewTransactionParser(c)

	// Process transactions with proper error handling
	var parsedTransactions []*models.Transaction
	for _, tx := range transactions {
		parsedTx, err := txParser.ProcessTransaction(ctx, tx)
		if err != nil {
			// Log error but continue processing other transactions
			fmt.Printf("Warning: Failed to parse transaction %s: %v\n", tx.Signature, err)
			continue
		}
		if parsedTx != nil {
			parsedTransactions = append(parsedTransactions, parsedTx)
		}
	}

	return parsedTransactions, nil
}

// SaveTransactions saves transactions to the database
func SaveTransactions(db *gorm.DB, walletID uint, transactions []*models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	// Use a transaction to ensure data consistency
	return db.Transaction(func(tx *gorm.DB) error {
		for _, transaction := range transactions {
			transaction.WalletID = walletID

			// Check for existing transaction to avoid duplicates
			var existing models.Transaction
			result := tx.Where("signature = ?", transaction.Signature).First(&existing)
			if result.Error == nil {
				// Transaction already exists, skip
				continue
			} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Unexpected error
				return fmt.Errorf("failed to check transaction %s: %w", transaction.Signature, result.Error)
			}

			// Save the transaction
			if err := tx.Create(transaction).Error; err != nil {
				return fmt.Errorf("failed to save transaction %s: %w", transaction.Signature, err)
			}
		}

		return nil
	})
}
