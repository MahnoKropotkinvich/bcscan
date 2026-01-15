package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/haswell/bcscan/internal/models"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
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

	return nil
}
