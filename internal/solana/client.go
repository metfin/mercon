package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wnt/mercon/internal/utils"
)

// Client represents a connection to the Solana blockchain
type Client struct {
	rpcClient *rpc.Client
	endpoint  string
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
func (c *Client) GetTransactionSigns(ctx context.Context, address string) ([]string, error) {
	client := utils.NewHTTPClient(
		utils.WithBaseURL(c.endpoint),
	)

	//We use this request, using a Helius rpc, this returns a thousand god damn transaction signatures.
	jsonParams, err := NewRpcBody("getSignaturesForAddress", []interface{}{
		address,
		map[string]interface{}{},
	})

	resp, err := client.Post("", jsonParams, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	var transactions struct {
		Result []string `json:"result"`
	}

	if err := json.Unmarshal(resp.Body, &transactions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transactions: %w", err)
	}

	return transactions.Result, nil
}

// GetTransaction returns a single transaction by signature
func (c *Client) GetTransaction(ctx context.Context, signature string) (*rpc.GetTransactionResult, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction signature: %w", err)
	}

	transaction, err := c.rpcClient.GetTransaction(
		ctx,
		sig,
		&rpc.GetTransactionOpts{
			Encoding:   solana.EncodingJSON,
			Commitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return transaction, nil
} 