package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wnt/mercon/internal/models"
	"github.com/wnt/mercon/internal/utils"
	"gorm.io/gorm"
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

// GetTransactionsInBulk returns a list of transactions for a list of signatures
func (c *Client) GetTransactionsInBulk(ctx context.Context, signatures []string) ([]*models.Transaction, error) {
	// Create a new HTTP client
	client := utils.NewHTTPClient()

	var transactions []*models.Transaction

	if len(signatures) == 0 {
		return transactions, nil
	}

	// For batch processing, we need to send individual requests
	// Process in reasonable batch sizes to avoid too large requests

	batchSize := 500
	for i := 0; i < len(signatures); i += batchSize {
		end := i + batchSize
		if end > len(signatures) {
			end = len(signatures)
		}

		fmt.Printf("Processing batch %d of %d\n", i/batchSize+1, len(signatures)/batchSize)

		currentBatch := signatures[i:end]
		var txRequests []map[string]interface{}

		for _, signature := range currentBatch {
			// Prepare the request body for each signature
			requestBody := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      signature, // Using signature as ID to match responses
				"method":  "getTransaction",
				"params": []interface{}{signature, map[string]interface{}{
					"encoding":                       "jsonParsed",
					"maxSupportedTransactionVersion": 0,
				}},
			}

			txRequests = append(txRequests, requestBody)
		}

		// Send the batch request
		resp, err := client.Post(c.endpoint, txRequests, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions: %w", err)
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("RPC request failed with status code: %d", resp.StatusCode)
		}

		// Define a proper structure to unmarshal the response
		var responses []RPCResponse

		if err := json.Unmarshal(resp.Body, &responses); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transactions: %w", err)
		}

		// Build a map of responses by ID for easier lookup
		responseMap := make(map[string]RPCResponse)
		for _, response := range responses {
			responseMap[response.ID] = response
		}

		// Process each signature in the order they were sent
		failedCount := 0
		for _, signature := range currentBatch {
			if response, ok := responseMap[signature]; ok && response.Error.Message == "" {
				transaction, err := convertToTransaction(response, signature)
				if err != nil {
					return nil, fmt.Errorf("failed to convert transaction %s: %w", signature, err)
				}
				transactions = append(transactions, transaction)
			} else {
				failedCount++
			}
		}

		fmt.Printf("Processed batch %d of %d\n", i/batchSize+1, len(signatures)/batchSize)
		fmt.Printf("Failed count: %d\n", failedCount)
		// Sleep for 1 second to avoid rate limiting
		time.Sleep(1 * time.Second)
	}

	return transactions, nil
}

// convertToTransaction converts an RPC response to our transaction model
func convertToTransaction(resp RPCResponse, signature string) (*models.Transaction, error) {
	// Extract transaction data
	tx := resp.Result

	// Create a new transaction from the RPC response
	transaction := &models.Transaction{
		Signature:            signature,
		BlockTime:            time.Unix(tx.BlockTime, 0),
		Fee:                  tx.Meta.Fee,
		FeeInSol:             float64(tx.Meta.Fee) / 1e9, // Convert lamports to SOL
		ComputeUnitsConsumed: uint64(tx.Meta.ComputeUnitsConsumed),
		RecentBlockhash:      tx.Transaction.Message.RecentBlockhash,
		InstructionsCount:    len(tx.Transaction.Message.Instructions),
		PriorityFee:          tx.Meta.PriorityFee,
		PriorityFeeInSol:     float64(tx.Meta.PriorityFee) / 1e9,
	}

	// Process signers
	var signers []string
	signersCount := 0

	for _, account := range tx.Transaction.Message.AccountKeys {
		if account.Signer {
			signers = append(signers, account.Pubkey)
			signersCount++
		}
	}

	transaction.Signers = signers
	transaction.SignersCount = signersCount

	// Store raw transaction data
	rawData, err := json.Marshal(resp)
	if err == nil {
		transaction.RawData = string(rawData)
	}

	// Create transaction instructions
	for idx, instruction := range tx.Transaction.Message.Instructions {
		txInstruction := &models.TransactionInstruction{
			Program:        instruction.Program,
			InstructionIdx: idx,
			ProgramID:      instruction.ProgramID,
			StackHeight:    &instruction.StackHeight,
		}

		// Handle parsed instruction if available
		if instruction.Parsed != nil {
			txInstruction.Type = instruction.Parsed.Type

			// Convert instruction info to JSON
			infoData, err := json.Marshal(instruction.Parsed.Info)
			if err == nil {
				txInstruction.Info = string(infoData)
			}
		}

		// Convert accounts to JSON
		accountsData, err := json.Marshal(instruction.Accounts)
		if err == nil {
			txInstruction.Accounts = string(accountsData)
		}

		// Convert data to JSON if present
		if instruction.Data != "" {
			txInstruction.Data = instruction.Data
		}

		transaction.Instructions = append(transaction.Instructions, *txInstruction)
	}

	// Process account keys
	for _, account := range tx.Transaction.Message.AccountKeys {
		txAccount := &models.TransactionAccount{
			Pubkey:   account.Pubkey,
			Signer:   account.Signer,
			Source:   account.Source,
			Writable: account.Writable,
		}

		transaction.AccountKeys = append(transaction.AccountKeys, *txAccount)
	}

	// Process inner instructions
	for _, innerInst := range tx.Meta.InnerInstructions {
		// Find parent instruction
		if innerInst.Index < len(transaction.Instructions) {
			parentInst := &transaction.Instructions[innerInst.Index]

			// Convert inner instructions to JSON
			innerData, err := json.Marshal(innerInst.Instructions)
			if err == nil {
				parentInst.InnerInstructions = string(innerData)
			}
		}
	}

	return transaction, nil
}

// GetAndProcessTransactions fetches transactions and processes them
func (c *Client) GetAndProcessTransactions(ctx context.Context, address string, filters Filters, db *gorm.DB) ([]*models.Transaction, error) {
	// First, get the transaction signatures
	signatures, err := c.GetTransactionSigns(ctx, address, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction signatures: %w", err)
	}

	// Fetch the transactions in bulk
	transactions, err := c.GetTransactionsInBulk(ctx, signatures)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions in bulk: %w", err)
	}

	//check if ANY of these txs are registered in the database. if they are then they're processed as well
	existingTxs := make(map[string]bool)
	db.Model(&models.Transaction{}).Where("signature IN ?", signatures).Pluck("signature", &existingTxs)

	// If any of these txs are registered in the database, remove them from the list
	transactions = utils.Filter(transactions, func(tx *models.Transaction) bool {
		return !existingTxs[tx.Signature]
	})

	// Create a parser to filter the transactions
	parser := NewTransactionParser(db, c)

	// Process the transactions
	for _, tx := range transactions {
		parser.ProcessTransaction(ctx, tx)
	}

	fmt.Printf("Processed %d transactions\n", len(transactions))

	return transactions, nil
}

// SaveTransactions saves transactions to the database
func SaveTransactions(db *gorm.DB, walletID uint, transactions []*models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	// Use a transaction to ensure data consistency
	return db.Transaction(func(tx *gorm.DB) error {
		for _, transaction := range transactions {
			// Set the wallet ID for all transactions
			transaction.WalletID = walletID

			// Create the transaction first
			if err := tx.Create(transaction).Error; err != nil {
				// Handle unique constraint violation (transaction already exists)
				if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
					// Log that we're skipping a duplicate transaction
					continue
				}
				return fmt.Errorf("failed to save transaction %s: %w", transaction.Signature, err)
			}

			// Save related instruction records
			for i := range transaction.Instructions {
				// Set the transaction ID for each instruction
				transaction.Instructions[i].TransactionID = transaction.ID
				if err := tx.Create(&transaction.Instructions[i]).Error; err != nil {
					return fmt.Errorf("failed to save instruction for transaction %s: %w", transaction.Signature, err)
				}
			}

			// Save related account records
			for i := range transaction.AccountKeys {
				// Set the transaction ID for each account
				transaction.AccountKeys[i].TransactionID = transaction.ID
				if err := tx.Create(&transaction.AccountKeys[i]).Error; err != nil {
					return fmt.Errorf("failed to save account for transaction %s: %w", transaction.Signature, err)
				}
			}
		}

		return nil
	})
}
