package models

import "time"

type RiskEvent struct {
	ID              int       `json:"id" db:"id"`
	EventType       string    `json:"event_type" db:"event_type"`
	Severity        string    `json:"severity" db:"severity"`
	ContractAddress string    `json:"contract_address" db:"contract_address"`
	TxHash          string    `json:"tx_hash" db:"tx_hash"`
	Description     string    `json:"description" db:"description"`
	Score           int       `json:"score" db:"score"`
	DetectedAt      time.Time `json:"detected_at" db:"detected_at"`
}
