package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/models"
	"go.uber.org/zap"
)

type TransactionRepository struct {
	db     *sql.DB
	redis  *cache.RedisClient
	logger *zap.Logger
}

func NewTransactionRepository(db *sql.DB, redis *cache.RedisClient, logger *zap.Logger) *TransactionRepository {
	return &TransactionRepository{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

// GetByHash 获取交易（先缓存后 DB）
func (r *TransactionRepository) GetByHash(ctx context.Context, txHash string) (*models.Transaction, error) {
	key := fmt.Sprintf("tx:%s", txHash)

	var tx models.Transaction
	err := r.redis.Get(ctx, key, &tx)
	if err == nil {
		return &tx, nil
	}

	query := `SELECT id, tx_hash, block_number, from_address, to_address, value, gas_price, gas_used, input_data, status, timestamp
	          FROM transactions WHERE tx_hash = $1`

	err = r.db.QueryRowContext(ctx, query, txHash).Scan(
		&tx.ID, &tx.TxHash, &tx.BlockNumber, &tx.FromAddress, &tx.ToAddress,
		&tx.Value, &tx.GasPrice, &tx.GasUsed, &tx.InputData, &tx.Status, &tx.Timestamp,
	)

	if err != nil {
		return nil, err
	}

	r.redis.Set(ctx, key, &tx, 1*time.Hour)
	return &tx, nil
}

func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (tx_hash, block_number, from_address, to_address, value, gas_price, gas_used, input_data, status, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tx_hash) DO NOTHING
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query,
		tx.TxHash,
		tx.BlockNumber,
		tx.FromAddress,
		tx.ToAddress,
		tx.Value,
		tx.GasPrice,
		tx.GasUsed,
		tx.InputData,
		tx.Status,
		tx.Timestamp,
	).Scan(&tx.ID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// 写入缓存
	key := fmt.Sprintf("tx:%s", tx.TxHash)
	r.redis.Set(ctx, key, tx, 1*time.Hour)

	return nil
}
