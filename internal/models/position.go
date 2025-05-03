package models

import (
	"time"

	"gorm.io/gorm"
)

// Position represents a wallet's position in a pool or token
type Position struct {
	gorm.Model
	WalletID        uint   `gorm:"index;not null"`
	PositionAddress string `gorm:"size:44;uniqueIndex;not null"`
	PoolAddress     string `gorm:"size:44;index;not null"`
	Owner           string `gorm:"size:44;index"`

	// Fee metrics
	FeeApr24h          *float64 `gorm:"default:0"` // Optional
	FeeApy24h          *float64 `gorm:"default:0"` // Optional
	DailyFeeYield      float64  `gorm:"default:0"`
	TotalFeeUSDClaimed float64  `gorm:"default:0"`
	TotalFeeXClaimed   int64    `gorm:"default:0"`
	TotalFeeYClaimed   int64    `gorm:"default:0"`

	// Reward metrics
	TotalRewardUSDClaimed float64 `gorm:"default:0"`
	TotalRewardXClaimed   int64   `gorm:"default:0"`
	TotalRewardYClaimed   int64   `gorm:"default:0"`

	// Timestamps
	LastUpdated time.Time `gorm:"index"`
	CreatedAt   time.Time `gorm:"index"`

	// JSON data
	Metadata string `gorm:"type:jsonb"`

	// Relationships - use indexes on foreign keys
	ClaimFees    []PositionClaimFee    `gorm:"foreignKey:PositionID"`
	ClaimRewards []PositionClaimReward `gorm:"foreignKey:PositionID"`
	Deposits     []PositionDeposit     `gorm:"foreignKey:PositionID"`
	Withdraws    []PositionWithdraw    `gorm:"foreignKey:PositionID"`
}

// Base struct for position-related records to reduce duplication
type PositionRecord struct {
	gorm.Model
	PositionID       uint      `gorm:"index;not null"`
	TransactionID    uint      `gorm:"index;not null"`
	PositionAddress  string    `gorm:"size:44;index;not null"`
	PairAddress      string    `gorm:"size:44;index"`
	OnchainTimestamp time.Time `gorm:"index"`
}

type PositionClaimFee struct {
	PositionRecord
	TokenXAmount    int64   `gorm:"default:0"`
	TokenXUSDAmount float64 `gorm:"default:0"`
	TokenYAmount    int64   `gorm:"default:0"`
	TokenYUSDAmount float64 `gorm:"default:0"`
}

type PositionClaimReward struct {
	PositionRecord
	RewardMintAddress string  `gorm:"size:44;index"`
	TokenAmount       int64   `gorm:"default:0"`
	TokenUSDAmount    float64 `gorm:"default:0"`
}

type PositionDeposit struct {
	PositionRecord
	ActiveBinID     int64   `gorm:"index"`
	Price           float64 `gorm:"default:0"`
	TokenXAmount    int64   `gorm:"default:0"`
	TokenXUSDAmount float64 `gorm:"default:0"`
	TokenYAmount    int64   `gorm:"default:0"`
	TokenYUSDAmount float64 `gorm:"default:0"`
}

type PositionWithdraw struct {
	PositionRecord
	ActiveBinID     int64   `gorm:"index"`
	Price           float64 `gorm:"default:0"`
	TokenXAmount    int64   `gorm:"default:0"`
	TokenXUSDAmount float64 `gorm:"default:0"`
	TokenYAmount    int64   `gorm:"default:0"`
	TokenYUSDAmount float64 `gorm:"default:0"`
}
