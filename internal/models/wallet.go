package models

import (
	"time"

	"gorm.io/gorm"
)

// Wallet represents a Solana wallet
type Wallet struct {
	gorm.Model
	Address            string    `gorm:"size:44;uniqueIndex;not null"`
	FirstTransactionAt time.Time `gorm:"index"`
	LastTransactionAt  time.Time `gorm:"index"`
	TransactionCount   int       `gorm:"default:0"`
	SOLBalance         float64   `gorm:"default:0"`
	LastScraped        time.Time

	// Relationships
	Transactions []Transaction `gorm:"foreignKey:WalletID"`
	Positions    []Position    `gorm:"foreignKey:WalletID"`
}
