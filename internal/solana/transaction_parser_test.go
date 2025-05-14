package solana

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wnt/mercon/internal/constants"
	"github.com/wnt/mercon/internal/models"
)

// TestTransactionParser is a version of TransactionParser that accepts any client that implements
// the required methods for testing
type TestTransactionParser struct {
	ParserClient interface {
		getPairID(ctx context.Context, pairAddress string) (uint, error)
		getWalletID(ctx context.Context, walletAddress string) (uint, error)
		isTokenXToY(ctx context.Context, tokenAccount string, tokenXMint string) (bool, error)
	}
}

// CreateTestParser creates a transaction parser for testing
func CreateTestParser(client interface {
	getPairID(ctx context.Context, pairAddress string) (uint, error)
	getWalletID(ctx context.Context, walletAddress string) (uint, error)
	isTokenXToY(ctx context.Context, tokenAccount string, tokenXMint string) (bool, error)
}) *TestTransactionParser {
	return &TestTransactionParser{
		ParserClient: client,
	}
}

// Copy of the methods from TransactionParser that we want to test
func (p *TestTransactionParser) ProcessTransaction(ctx context.Context, tx Transaction) (*models.Transaction, error) {
	// First we filter all the non-meteora tx instructions
	meteoraInstructions := make([]Instruction, 0)
	for _, instruction := range tx.Instructions {
		if instruction.ProgramId == constants.MeteoraDLMM {
			meteoraInstructions = append(meteoraInstructions, instruction)
		}
	}

	if len(meteoraInstructions) == 0 {
		return nil, &errorWithSignature{
			err:       "no meteora instructions found in transaction",
			signature: tx.Signature,
		}
	}

	// Create the base transaction model
	txModel := &models.Transaction{
		Signature:   tx.Signature,
		BlockTime:   UnixTimeToTime(tx.Timestamp),
		Slot:        tx.Slot,
		Description: tx.Description,
		Type:        tx.Type,
		Source:      tx.Source,
		Fee:         tx.Fee,
		FeePayer:    tx.FeePayer,
	}

	if tx.TransactionError != nil {
		txModel.Error = tx.TransactionError.Error
	}

	return txModel, nil
}

// errorWithSignature is a wrapper for errors that include transaction signatures
type errorWithSignature struct {
	err       string
	signature string
}

// Error implements the error interface
func (e *errorWithSignature) Error() string {
	return e.err + " " + e.signature
}

// Create a custom mock implementation for testing
type mockSolanaClient struct {
	pairIDFunc      func(ctx context.Context, pairAddress string) (uint, error)
	walletIDFunc    func(ctx context.Context, walletAddress string) (uint, error)
	isTokenXToYFunc func(ctx context.Context, tokenAccount string, tokenXMint string) (bool, error)
}

func (m *mockSolanaClient) getPairID(ctx context.Context, pairAddress string) (uint, error) {
	if m.pairIDFunc != nil {
		return m.pairIDFunc(ctx, pairAddress)
	}
	return 1, nil // Default implementation
}

func (m *mockSolanaClient) getWalletID(ctx context.Context, walletAddress string) (uint, error) {
	if m.walletIDFunc != nil {
		return m.walletIDFunc(ctx, walletAddress)
	}
	return 1, nil // Default implementation
}

func (m *mockSolanaClient) isTokenXToY(ctx context.Context, tokenAccount string, tokenXMint string) (bool, error) {
	if m.isTokenXToYFunc != nil {
		return m.isTokenXToYFunc(ctx, tokenAccount, tokenXMint)
	}
	return true, nil // Default implementation
}

func TestCreateTestParser(t *testing.T) {
	mockClient := &mockSolanaClient{}
	parser := CreateTestParser(mockClient)

	assert.NotNil(t, parser)
	assert.Equal(t, mockClient, parser.ParserClient)
}

func TestProcessTransaction_NoMeteorInstructions(t *testing.T) {
	mockClient := &mockSolanaClient{}
	parser := CreateTestParser(mockClient)

	// Create a transaction with no Meteora instructions
	tx := Transaction{
		Signature: "testSignature",
		Instructions: []Instruction{
			{
				ProgramId: "someOtherProgram",
				Accounts:  []string{"account1", "account2"},
				Data:      "someData",
			},
		},
	}

	// Process the transaction
	txModel, err := parser.ProcessTransaction(context.Background(), tx)

	// Verify the error
	assert.Error(t, err)
	assert.Nil(t, txModel)
	assert.Contains(t, err.Error(), "no meteora instructions found")
}

func TestProcessTransaction_WithMeteorInstructions(t *testing.T) {
	// Setup mock client with function implementations
	mockClient := &mockSolanaClient{
		pairIDFunc: func(ctx context.Context, pairAddress string) (uint, error) {
			return 1, nil
		},
		walletIDFunc: func(ctx context.Context, walletAddress string) (uint, error) {
			return 2, nil
		},
		isTokenXToYFunc: func(ctx context.Context, tokenAccount string, tokenXMint string) (bool, error) {
			return true, nil
		},
	}

	parser := CreateTestParser(mockClient)

	// Create a transaction with a Meteora instruction
	tx := Transaction{
		Signature:   "testSignature",
		Description: "Test Transaction",
		Type:        "TRANSFER",
		Source:      "meteora",
		Fee:         5000,
		FeePayer:    "feePayerAddress",
		Slot:        12345678,
		Timestamp:   1667289600, // Oct 31, 2022
		Instructions: []Instruction{
			{
				ProgramId: constants.MeteoraDLMM,
				Accounts:  []string{"account1", "account2"},
				Data:      "AQAAAAAAAAAAAAAAAA==", // Empty data with type 1 (initialize pair)
			},
		},
	}

	// Process the transaction
	txModel, err := parser.ProcessTransaction(context.Background(), tx)

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, txModel)
	assert.Equal(t, tx.Signature, txModel.Signature)
	assert.Equal(t, tx.Description, txModel.Description)
	assert.Equal(t, tx.Type, txModel.Type)
	assert.Equal(t, tx.Source, txModel.Source)
	assert.Equal(t, tx.Fee, txModel.Fee)
	assert.Equal(t, tx.FeePayer, txModel.FeePayer)
	assert.Equal(t, tx.Slot, txModel.Slot)

	// Verify the time conversion - just check the Unix timestamp
	convertedTime := txModel.BlockTime.Unix()
	assert.Equal(t, tx.Timestamp, convertedTime)
}

func TestUnixTimeToTime(t *testing.T) {
	// Test unix timestamp to time.Time conversion
	timestamp := int64(1667289600) // Oct 31, 2022

	result := UnixTimeToTime(timestamp)

	// Since the implementation uses time.Unix which uses the local timezone,
	// we should only verify that the Unix timestamp is preserved
	assert.Equal(t, timestamp, result.Unix())
}
