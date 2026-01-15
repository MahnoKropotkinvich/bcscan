package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/haswell/bcscan/internal/models"
)

type BlockRepository struct {
	db *sql.DB
}

func NewBlockRepository(db *sql.DB) *BlockRepository {
	return &BlockRepository{db: db}
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

	return nil
}
