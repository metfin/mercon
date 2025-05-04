package models

import (
	"time"

	"gorm.io/gorm"
)

// MeteoraPair represents a Meteora DLMM liquidity pair
type MeteoraPair struct {
	gorm.Model
	Address    string `gorm:"size:44;uniqueIndex;not null"`
	TokenMintX string `gorm:"size:44;index"`
	TokenMintY string `gorm:"size:44;index"`
	ReserveX   string `gorm:"size:44"`
	ReserveY   string `gorm:"size:44"`
	Oracle     string `gorm:"size:44"`
	ActiveID   int32
	BinStep    uint16
	Status     string `gorm:"size:20;default:'active'"`

	// Relationships
	Positions []MeteoraPosition `gorm:"foreignKey:PairID"`
	Swaps     []MeteoraSwap     `gorm:"foreignKey:PairID"`
	Rewards   []MeteoraReward   `gorm:"foreignKey:PairID"`
}

// MeteoraPosition represents a liquidity position in a Meteora DLMM pair
type MeteoraPosition struct {
	gorm.Model
	Address    string `gorm:"size:44;uniqueIndex;not null"`
	PairID     uint   `gorm:"index;not null"`
	WalletID   uint   `gorm:"index;not null"`
	Owner      string `gorm:"size:44;index"`
	LowerBinID int32
	Width      int32
	CreatedAt  time.Time
	ClosedAt   *time.Time
	Status     string `gorm:"size:20;default:'active'"`

	// Relationships
	LiquidityAdditions []MeteoraLiquidityAddition `gorm:"foreignKey:PositionID"`
	LiquidityRemovals  []MeteoraLiquidityRemoval  `gorm:"foreignKey:PositionID"`
	FeeClaims          []MeteoraFeeClaim          `gorm:"foreignKey:PositionID"`
	RewardClaims       []MeteoraRewardClaim       `gorm:"foreignKey:PositionID"`
	Wallet             Wallet                     `gorm:"foreignKey:WalletID"`
}

// MeteoraSwap represents a token swap in a Meteora DLMM pair
type MeteoraSwap struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	PairID        uint   `gorm:"index;not null"`
	WalletID      uint   `gorm:"index;not null"`
	User          string `gorm:"size:44;index"`
	TokenInMint   string `gorm:"size:44"`
	TokenOutMint  string `gorm:"size:44"`
	AmountIn      uint64
	AmountOut     uint64
	MinAmountOut  uint64
	Fee           uint64
	ProtocolFee   uint64
	FeeBps        uint16
	SwapTime      time.Time `gorm:"index"`
	StartBinID    int32
	EndBinID      int32
	SwapForY      bool // true if X->Y, false if Y->X

	// Relationships
	Transaction Transaction `gorm:"foreignKey:TransactionID"`
	Wallet      Wallet      `gorm:"foreignKey:WalletID"`
}

// MeteoraLiquidityAddition represents adding liquidity to a position
type MeteoraLiquidityAddition struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	PositionID    uint   `gorm:"index;not null"`
	PairID        uint   `gorm:"index;not null"`
	WalletID      uint   `gorm:"index;not null"`
	User          string `gorm:"size:44;index"`
	AmountX       uint64
	AmountY       uint64
	ActiveID      int32
	AddTime       time.Time `gorm:"index"`

	// Store distribution of liquidity to bins
	BinDistribution string `gorm:"type:jsonb"`

	// Relationships
	Transaction Transaction     `gorm:"foreignKey:TransactionID"`
	Wallet      Wallet          `gorm:"foreignKey:WalletID"`
	Position    MeteoraPosition `gorm:"foreignKey:PositionID"`
	Pair        MeteoraPair     `gorm:"foreignKey:PairID"`
}

// MeteoraLiquidityRemoval represents removing liquidity from a position
type MeteoraLiquidityRemoval struct {
	gorm.Model
	TransactionID uint      `gorm:"index;not null"`
	PositionID    uint      `gorm:"index;not null"`
	PairID        uint      `gorm:"index;not null"`
	WalletID      uint      `gorm:"index;not null"`
	User          string    `gorm:"size:44;index"`
	RemoveTime    time.Time `gorm:"index"`

	// Store which bins had liquidity removed
	BinReductions string `gorm:"type:jsonb"`

	// Relationships
	Transaction Transaction     `gorm:"foreignKey:TransactionID"`
	Wallet      Wallet          `gorm:"foreignKey:WalletID"`
	Position    MeteoraPosition `gorm:"foreignKey:PositionID"`
	Pair        MeteoraPair     `gorm:"foreignKey:PairID"`
}

// MeteoraFeeClaim represents claiming accumulated fees
type MeteoraFeeClaim struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	PositionID    uint   `gorm:"index;not null"`
	PairID        uint   `gorm:"index;not null"`
	WalletID      uint   `gorm:"index;not null"`
	User          string `gorm:"size:44;index"`
	AmountX       uint64
	AmountY       uint64
	ClaimTime     time.Time `gorm:"index"`

	// Relationships
	Transaction Transaction     `gorm:"foreignKey:TransactionID"`
	Wallet      Wallet          `gorm:"foreignKey:WalletID"`
	Position    MeteoraPosition `gorm:"foreignKey:PositionID"`
	Pair        MeteoraPair     `gorm:"foreignKey:PairID"`
}

// MeteoraReward represents a reward for liquidity providers
type MeteoraReward struct {
	gorm.Model
	PairID         uint `gorm:"index;not null"`
	RewardIndex    uint64
	RewardVault    string `gorm:"size:44"`
	RewardMint     string `gorm:"size:44"`
	Funder         string `gorm:"size:44"`
	RewardDuration uint64
	StartTime      time.Time
	EndTime        time.Time
	Status         string `gorm:"size:20;default:'active'"`

	// Relationships
	Pair MeteoraPair `gorm:"foreignKey:PairID"`
}

// MeteoraRewardFunding represents funding a reward
type MeteoraRewardFunding struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	RewardID      uint   `gorm:"index;not null"`
	PairID        uint   `gorm:"index;not null"`
	WalletID      uint   `gorm:"index;not null"`
	Funder        string `gorm:"size:44;index"`
	Amount        uint64
	CarryForward  bool
	FundTime      time.Time `gorm:"index"`

	// Relationships
	Transaction Transaction   `gorm:"foreignKey:TransactionID"`
	Wallet      Wallet        `gorm:"foreignKey:WalletID"`
	Reward      MeteoraReward `gorm:"foreignKey:RewardID"`
	Pair        MeteoraPair   `gorm:"foreignKey:PairID"`
}

// MeteoraRewardClaim represents claiming rewards
type MeteoraRewardClaim struct {
	gorm.Model
	TransactionID uint   `gorm:"index;not null"`
	PositionID    uint   `gorm:"index;not null"`
	RewardID      uint   `gorm:"index;not null"`
	PairID        uint   `gorm:"index;not null"`
	WalletID      uint   `gorm:"index;not null"`
	User          string `gorm:"size:44;index"`
	Amount        uint64
	ClaimTime     time.Time `gorm:"index"`

	// Relationships
	Transaction Transaction     `gorm:"foreignKey:TransactionID"`
	Wallet      Wallet          `gorm:"foreignKey:WalletID"`
	Position    MeteoraPosition `gorm:"foreignKey:PositionID"`
	Reward      MeteoraReward   `gorm:"foreignKey:RewardID"`
	Pair        MeteoraPair     `gorm:"foreignKey:PairID"`
}
