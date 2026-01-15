package models

import "time"

type Block struct {
	ID               int64     `json:"id" db:"id"`
	BlockNumber      int64     `json:"block_number" db:"block_number"`
	BlockHash        string    `json:"block_hash" db:"block_hash"`
	ParentHash       string    `json:"parent_hash" db:"parent_hash"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	Miner            string    `json:"miner" db:"miner"`
	GasUsed          int64     `json:"gas_used" db:"gas_used"`
	GasLimit         int64     `json:"gas_limit" db:"gas_limit"`
	TransactionCount int       `json:"transaction_count" db:"transaction_count"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
