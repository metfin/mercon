package solana

import (
	"context"
	"encoding/json"
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

// Client represents a connection to the Solana blockchain
type Client struct {
	rpcClient  *rpc.Client
	endpoint   string
	httpClient *utils.HTTPClient
}

type Filters struct {
	After  time.Time
	LastTx string
}

// NewClient creates a new Solana client
func NewClient() (*Client, error) {
	endpoint := os.Getenv("RPC_URL")
	if endpoint == "" {
		return nil, fmt.Errorf("RPC_URL environment variable is not set")
	}

	rpcClient := rpc.New(endpoint)

	// Check connection by getting the latest block height
	_, err := rpcClient.GetBlockHeight(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Solana RPC: %w", err)
	}

	// Create HTTP client for API requests
	httpClient := utils.NewHTTPClient(
		utils.WithTimeout(30*time.Second),
		utils.WithBaseURL(constants.HeliusBaseURL),
	)

	return &Client{
		rpcClient:  rpcClient,
		endpoint:   endpoint,
		httpClient: httpClient,
	}, nil
}

func (c *Client) GetTransactions(ctx context.Context, address string, filters Filters) ([]Transaction, error) {
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("HELIUS_API_KEY environment variable is not set")
	}

	// Build query parameters
	queryParams := map[string]string{
		"api-key": apiKey,
	}

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

func (c *Client) GetAndParseTransactions(ctx context.Context, address string, filters Filters) ([]*models.Transaction, error) {
	transactions, err := c.GetTransactions(ctx, address, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	txParser := NewTransactionParser(c)

	parsedTransactions := make([]*models.Transaction, len(transactions))
	for i, tx := range transactions {
		parsedTransactions[i], err = txParser.ProcessTransaction(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transaction %s: %w", tx.Signature, err)
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
			if err := tx.Create(transaction).Error; err != nil {
				return fmt.Errorf("failed to save transaction %s: %w", transaction.Signature, err)
			}
		}

		return nil
	})
}
