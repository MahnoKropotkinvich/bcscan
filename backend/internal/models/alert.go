package models

import "time"

// Alert 告警模型
type Alert struct {
	ID             int64      `json:"id"`
	RiskEventID    int64      `json:"risk_event_id"`
	Title          string     `json:"title"`
	Message        string     `json:"message"`
	Severity       string     `json:"severity"`
	Status         string     `json:"status"`
	AssignedTo     *int64     `json:"assigned_to,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *int64     `json:"acknowledged_by,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     *int64     `json:"resolved_by,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`

	// 关联的风险事件信息
	TxHash          string  `json:"tx_hash,omitempty"`
	ContractAddress string  `json:"contract_address,omitempty"`
	Score           float64 `json:"score,omitempty"`
}

// AlertHistory 告警处理历史
type AlertHistory struct {
	ID        int64     `json:"id"`
	AlertID   int64     `json:"alert_id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Action    string    `json:"action"`
	OldStatus string    `json:"old_status,omitempty"`
	NewStatus string    `json:"new_status,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
