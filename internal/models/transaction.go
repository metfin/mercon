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
	Fee                  int64     // Store in lamports instead of float64
	FeeInSol             float64
	PriorityFee          int64 // Store in lamports instead of float64
	PriorityFeeInSol     float64
	Confirmations        uint64
	ComputeUnitsConsumed uint64
	RecentBlockhash      string   `gorm:"size:88"`
	InstructionsCount    int      `gorm:"default:0"`
	SignersCount         int      `gorm:"default:0"`
	Signers              []string `gorm:"type:jsonb"`
	WalletID             uint     `gorm:"index;not null"`
	RawData              string   `gorm:"type:jsonb"`

	// Relationships
	Instructions []TransactionInstruction `gorm:"foreignKey:TransactionID"`
	AccountKeys  []TransactionAccount     `gorm:"foreignKey:TransactionID"`
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
}

// TransactionAccount represents an account involved in a transaction
type TransactionAccount struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	Pubkey        string `gorm:"size:44;index"`
	Signer        bool   `gorm:"index"`
	Source        string `gorm:"size:20"`
	Writable      bool   `gorm:"index"`
}
