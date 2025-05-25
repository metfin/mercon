package models

import (
	"time"

	"gorm.io/gorm"
)

// Transaction represents a Solana blockchain transaction
type Transaction struct {
	gorm.Model
	Signature   string    `gorm:"size:88;uniqueIndex;not null"`
	WalletID    uint      `gorm:"index;not null"`
	BlockTime   time.Time `gorm:"index"`
	Slot        int64     `gorm:"index"`
	Description string
	Type        string `gorm:"size:50;index"`
	Source      string `gorm:"size:50;index"`
	Fee         int64
	FeePayer    string `gorm:"size:44;index"`
	Error       string

	// Transaction data - embed important data directly in transaction to reduce table count
	HasNativeTransfers bool
	HasTokenTransfers  bool
	HasInstructions    bool

	// Relationships
	Wallet          Wallet                   `gorm:"foreignKey:WalletID"`
	Instructions    []TransactionInstruction `gorm:"foreignKey:TransactionID"`
	NativeTransfers []NativeTransfer         `gorm:"foreignKey:TransactionID"`
	TokenTransfers  []TokenTransfer          `gorm:"foreignKey:TransactionID"`

	// Protocol-specific relationships (from meteora.go)
	Swaps              []MeteoraSwap              `gorm:"foreignKey:TransactionID"`
	LiquidityAdditions []MeteoraLiquidityAddition `gorm:"foreignKey:TransactionID"`
	LiquidityRemovals  []MeteoraLiquidityRemoval  `gorm:"foreignKey:TransactionID"`
	FeeClaims          []MeteoraFeeClaim          `gorm:"foreignKey:TransactionID"`
	RewardClaims       []MeteoraRewardClaim       `gorm:"foreignKey:TransactionID"`
	RewardFundings     []MeteoraRewardFunding     `gorm:"foreignKey:TransactionID"`
}

// TransactionInstruction represents an instruction in a transaction
type TransactionInstruction struct {
	gorm.Model
	TransactionID    uint   `gorm:"index;not null"`
	ProgramID        string `gorm:"size:44;index"`
	InstructionIndex int    `gorm:"index"`
	Data             string `gorm:"type:text"`
	IsInner          bool
	ParentIndex      *int // For inner instructions, index of parent instruction

	// Relationships
	Transaction Transaction          `gorm:"foreignKey:TransactionID"`
	Accounts    []TransactionAccount `gorm:"foreignKey:InstructionID"`
}

// TransactionAccount represents an account used in a transaction
type TransactionAccount struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	InstructionID *uint  `gorm:"index"` // Optional, as some accounts are only transaction level
	Pubkey        string `gorm:"size:44;index"`
	Signer        bool   `gorm:"index"`
	Writable      bool
	ProgramOwned  bool
	AccountIndex  int // Position in the transaction or instruction accounts array

	// Relationships
	Transaction Transaction             `gorm:"foreignKey:TransactionID"`
	Instruction *TransactionInstruction `gorm:"foreignKey:InstructionID"`
}

// NativeTransfer represents a SOL transfer in a transaction
type NativeTransfer struct {
	gorm.Model
	TransactionID   uint   `gorm:"index;not null"`
	FromUserAccount string `gorm:"size:44;index"`
	ToUserAccount   string `gorm:"size:44;index"`
	Amount          int64
	AmountDecimal   float64 `gorm:"type:decimal(20,9)"` // SOL amount with decimals

	// Relationships
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
}

// TokenTransfer represents an SPL token transfer in a transaction
type TokenTransfer struct {
	gorm.Model
	TransactionID    uint   `gorm:"index;not null"`
	FromUserAccount  string `gorm:"size:44;index"`
	ToUserAccount    string `gorm:"size:44;index"`
	FromTokenAccount string `gorm:"size:44;index"`
	ToTokenAccount   string `gorm:"size:44;index"`
	TokenAmount      int64
	Mint             string `gorm:"size:44;index"`
	Decimals         int
	AmountDecimal    float64 `gorm:"type:decimal(30,15)"` // Token amount with decimals

	// Optional USD value fields
	USDValue   float64
	TokenPrice float64

	// Relationships
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
}

// SwapEvent represents token swap events (simplified)
type SwapEvent struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	Source        string `gorm:"size:50;index"`
	Program       string `gorm:"size:44;index"`

	// Token input
	TokenInMint    string `gorm:"size:44;index"`
	TokenInAccount string `gorm:"size:44"`
	TokenInAmount  int64
	TokenInUSD     float64

	// Token output
	TokenOutMint    string `gorm:"size:44;index"`
	TokenOutAccount string `gorm:"size:44"`
	TokenOutAmount  int64
	TokenOutUSD     float64

	// Fee information
	FeeAmount int64
	FeeUSD    float64

	// Relationships
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
}

// CompressionEvent represents compressed account operations (simplified)
type CompressionEvent struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	Type          string `gorm:"size:50;index"`
	TreeID        string `gorm:"size:44;index"`
	AssetID       string `gorm:"size:64;index"`
	NewOwner      string `gorm:"size:44;index"`
	OldOwner      string `gorm:"size:44;index"`

	// For rewards
	RewardAmount int64

	// For authority changes
	Account string `gorm:"size:44;index"`
	From    string `gorm:"size:44;index"`
	To      string `gorm:"size:44;index"`

	// Relationships
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
}
