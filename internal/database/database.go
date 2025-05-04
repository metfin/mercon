package database

import (
	"fmt"
	"os"
	"time"

	"github.com/wnt/mercon/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	// Configure GORM with optimized settings
	config := &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Silent),
		PrepareStmt: true, // Prepare statement for better performance
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Set connection pool limits
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Migrate database schema
	if err := migrateSchema(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrateSchema(db *gorm.DB) error {
	// Migrate models
	if err := db.AutoMigrate(
		&models.Wallet{},
		&models.Transaction{},
		&models.TransactionInstruction{},
		&models.TransactionAccount{},
		&models.MeteoraPosition{},
		&models.MeteoraSwap{},
		&models.MeteoraLiquidityAddition{},
		&models.MeteoraLiquidityRemoval{},
		&models.MeteoraFeeClaim{},
		&models.MeteoraReward{},
		&models.MeteoraRewardFunding{},
		&models.MeteoraRewardClaim{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Add composite indexes for common query patterns
	db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_wallet_blocktime ON transactions(wallet_id, block_time)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_position_wallet_pool ON positions(wallet_id, pool_address)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_transaction_accounts_pubkey_signer ON transaction_accounts(pubkey, signer) WHERE signer = true")

	return nil
}
