package models

import "time"

type Event struct {
	ID              int64     `json:"id" db:"id"`
	TxHash          string    `json:"tx_hash" db:"tx_hash"`
	ContractAddress string    `json:"contract_address" db:"contract_address"`
	EventName       string    `json:"event_name" db:"event_name"`
	EventSignature  string    `json:"event_signature" db:"event_signature"`
	Topics          string    `json:"topics" db:"topics"`
	Data            string    `json:"data" db:"data"`
	LogIndex        int       `json:"log_index" db:"log_index"`
	DecodedData     string    `json:"decoded_data" db:"decoded_data"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
