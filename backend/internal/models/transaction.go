package models

import "time"

type Transaction struct {
	ID          int64     `json:"id" db:"id"`
	TxHash      string    `json:"tx_hash" db:"tx_hash"`
	BlockNumber int64     `json:"block_number" db:"block_number"`
	FromAddress string    `json:"from_address" db:"from_address"`
	ToAddress   string    `json:"to_address" db:"to_address"`
	Value       string    `json:"value" db:"value"`
	GasPrice    int64     `json:"gas_price" db:"gas_price"`
	GasUsed     int64     `json:"gas_used" db:"gas_used"`
	InputData   string    `json:"input_data" db:"input_data"`
	Status      int16     `json:"status" db:"status"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
