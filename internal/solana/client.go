package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wnt/mercon/internal/utils"
)

// Client represents a connection to the Solana blockchain
type Client struct {
	rpcClient *rpc.Client
	endpoint  string
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

	return &Client{
		rpcClient: rpcClient,
		endpoint:  endpoint,
	}, nil
}

// GetTransactions returns the recent transactions for a wallet address
func (c *Client) GetTransactionSigns(ctx context.Context, address string, filters Filters) ([]string, error) {
	var allSignatures []string
	lastTx := filters.LastTx
	afterTime := filters.After
	maxLoops := 10
	loopCount := 0

	// Create HTTP client
	client := utils.NewHTTPClient()

	// Use a bounded loop to fetch transactions in batches of up to 1000
	for loopCount < maxLoops {
		loopCount++

		// Prepare the request body directly
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      "mercon-client",
			"method":  "getSignaturesForAddress",
			"params":  []interface{}{address, map[string]interface{}{}},
		}

		// If we have a last transaction signature, add it to the params
		if lastTx != "" {
			requestBody["params"] = []interface{}{address, map[string]interface{}{
				"before": lastTx,
			}}
		}

		// Make the HTTP request
		resp, err := client.Post(c.endpoint, requestBody, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions: %w", err)
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("RPC request failed with status code: %d", resp.StatusCode)
		}

		// Define a proper structure to unmarshal the response
		var response struct {
			ID      string `json:"id"`
			JSONRPC string `json:"jsonrpc"`
			Result  []struct {
				Signature string `json:"signature"`
				BlockTime int64  `json:"blockTime"` // Using int64 for Unix timestamp
			} `json:"result"`
		}

		if err := json.Unmarshal(resp.Body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transactions: %w", err)
		}

		// If no results, we're done
		if len(response.Result) == 0 {
			break
		}

		// Extract signatures from the current batch, filtering by After time if specified
		var batchSignatures []string
		shouldContinue := true

		for _, tx := range response.Result {
			// Convert Unix timestamp to time.Time
			txTime := time.Unix(tx.BlockTime, 0)

			if !afterTime.IsZero() && txTime.Before(afterTime) {
				// We've reached transactions older than our filter time, no need to fetch more
				shouldContinue = false
				break
			}
			batchSignatures = append(batchSignatures, tx.Signature)
		}

		// If no valid signatures after filtering, we're done
		if len(batchSignatures) == 0 {
			break
		}

		// Append current batch to our results
		allSignatures = append(allSignatures, batchSignatures...)

		// If the batch size is less than 1000, we've reached the end
		if len(response.Result) < 1000 {
			break
		}

		// If we shouldn't continue or there are no more results, exit the loop
		if !shouldContinue {
			break
		}

		// Set up for the next batch
		lastTx = response.Result[len(response.Result)-1].Signature
	}

	return allSignatures, nil
}
