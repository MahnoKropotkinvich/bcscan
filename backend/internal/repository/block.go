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

type BlockRepository struct {
	db     *sql.DB
	redis  *cache.RedisClient
	logger *zap.Logger
}

func NewBlockRepository(db *sql.DB, redis *cache.RedisClient, logger *zap.Logger) *BlockRepository {
	return &BlockRepository{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

// GetByNumber 获取区块（先缓存后 DB）
func (r *BlockRepository) GetByNumber(ctx context.Context, blockNumber int64) (*models.Block, error) {
	key := fmt.Sprintf("block:%d", blockNumber)

	var block models.Block
	err := r.redis.Get(ctx, key, &block)
	if err == nil {
		return &block, nil
	}

	query := `SELECT id, block_number, block_hash, parent_hash, timestamp, miner, gas_used, gas_limit, transaction_count
	          FROM blocks WHERE block_number = $1`

	err = r.db.QueryRowContext(ctx, query, blockNumber).Scan(
		&block.ID, &block.BlockNumber, &block.BlockHash, &block.ParentHash,
		&block.Timestamp, &block.Miner, &block.GasUsed, &block.GasLimit, &block.TransactionCount,
	)

	if err != nil {
		return nil, err
	}

	r.redis.Set(ctx, key, &block, 24*time.Hour)
	return &block, nil
}

func (r *BlockRepository) Create(ctx context.Context, block *models.Block) error {
	query := `
		INSERT INTO blocks (block_number, block_hash, parent_hash, timestamp, miner, gas_used, gas_limit, transaction_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (block_number) DO NOTHING
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query,
		block.BlockNumber,
		block.BlockHash,
		block.ParentHash,
		block.Timestamp,
		block.Miner,
		block.GasUsed,
		block.GasLimit,
		block.TransactionCount,
	).Scan(&block.ID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to insert block: %w", err)
	}

	// 写入缓存（区块不可变，长期缓存）
	key := fmt.Sprintf("block:%d", block.BlockNumber)
	r.redis.Set(ctx, key, block, 24*time.Hour)

	return nil
}
