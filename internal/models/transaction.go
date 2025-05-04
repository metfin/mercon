package models

import (
	"time"

	"gorm.io/gorm"
)

// Transaction represents a Solana blockchain transaction
type Transaction struct {
	gorm.Model
	Signature            string    `gorm:"size:88;uniqueIndex;not null"`
	BlockTime            time.Time `gorm:"index"`
	Fee                  int64     // Store in lamports
	PriorityFee          int64     // Store in lamports
	Confirmations        uint64
	ComputeUnitsConsumed uint64
	RecentBlockhash      string   `gorm:"size:88"`
	InstructionsCount    int      `gorm:"default:0"`
	SignersCount         int      `gorm:"default:0"`
	Signers              []string `gorm:"type:jsonb"`
	WalletID             uint     `gorm:"index;not null"`
	RawData              string   `gorm:"type:jsonb"`

	// Relationships
	Instructions        []TransactionInstruction   `gorm:"foreignKey:TransactionID"`
	AccountKeys         []TransactionAccount       `gorm:"foreignKey:TransactionID"`
	MeteoraSwaps        []MeteoraSwap              `gorm:"foreignKey:TransactionID"`
	MeteoraAdditions    []MeteoraLiquidityAddition `gorm:"foreignKey:TransactionID"`
	MeteoraRemovals     []MeteoraLiquidityRemoval  `gorm:"foreignKey:TransactionID"`
	MeteoraFeeClaims    []MeteoraFeeClaim          `gorm:"foreignKey:TransactionID"`
	MeteoraRewardClaims []MeteoraRewardClaim       `gorm:"foreignKey:TransactionID"`
	MeteoraRewardFunds  []MeteoraRewardFunding     `gorm:"foreignKey:TransactionID"`
	Wallet              Wallet                     `gorm:"foreignKey:WalletID"`
}

// TransactionInstruction represents instruction-specific details within a transaction
type TransactionInstruction struct {
	gorm.Model
	TransactionID     uint   `gorm:"index;not null"`
	Program           string `gorm:"size:44;index"`
	InstructionIdx    int    `gorm:"index"`
	ProgramID         string `gorm:"size:44;index"`
	Type              string `gorm:"size:100;index"`
	Accounts          string `gorm:"type:jsonb"`
	Data              string `gorm:"type:jsonb"`
	InnerInstructions string `gorm:"type:jsonb"`
	Info              string `gorm:"type:jsonb"`
	StackHeight       *int   // Using pointer to handle null values

	// Relationship
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
}

// TransactionAccount represents an account involved in a transaction
type TransactionAccount struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	Pubkey        string `gorm:"size:44;index"`
	Signer        bool   `gorm:"index"`
	Source        string `gorm:"size:20"`
	Writable      bool   `gorm:"index"`

	// Relationship
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
}

// Helper methods to convert between lamports and SOL
func (t *Transaction) FeeInSOL() float64 {
	return float64(t.Fee) / 1000000000.0
}

func (t *Transaction) PriorityFeeInSOL() float64 {
	return float64(t.PriorityFee) / 1000000000.0
}

// BeforeSave hook to calculate SOL values before saving
func (t *Transaction) BeforeSave(tx *gorm.DB) error {
	// These will be used for display/querying but aren't stored directly
	// Will be calculated on-the-fly by the helper methods above
	return nil
}
