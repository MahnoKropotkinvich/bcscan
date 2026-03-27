package models

import (
	"encoding/json"
	"time"
)

// ExplorerTransaction 交易浏览器数据模型
type ExplorerTransaction struct {
	TxHash           string          `json:"tx_hash"`
	BlockNumber      int64           `json:"block_number"`
	FromAddress      string          `json:"from_address"`
	ToAddress        string          `json:"to_address"`
	Value            string          `json:"value"`
	GasPrice         int64           `json:"gas_price"`
	GasUsed          int64           `json:"gas_used"`
	GasLimit         *int64          `json:"gas_limit,omitempty"`
	InputData        *string         `json:"input_data,omitempty"`
	FunctionSelector *string         `json:"function_selector,omitempty"`
	FunctionName     *string         `json:"function_name,omitempty"`
	FunctionDesc     *string         `json:"function_desc,omitempty"`
	Status           int16           `json:"status"`
	Timestamp        time.Time       `json:"timestamp"`
	CallStack        json.RawMessage `json:"call_stack,omitempty"`
	EventsData       json.RawMessage `json:"events_data,omitempty"`
	RiskCount        int             `json:"risk_count"`
}

// TxSummary 交易摘要（地址关联交易列表用）
type TxSummary struct {
	TxHash           string    `json:"tx_hash"`
	BlockNumber      int64     `json:"block_number"`
	FromAddress      string    `json:"from_address"`
	ToAddress        string    `json:"to_address"`
	Value            string    `json:"value"`
	GasUsed          int64     `json:"gas_used"`
	FunctionSelector *string   `json:"function_selector,omitempty"`
	FunctionName     *string   `json:"function_name,omitempty"`
	Status           int16     `json:"status"`
	Timestamp        time.Time `json:"timestamp"`
}

// TxBrief 交易简要（最近交易列表用）
type TxBrief struct {
	TxHash           string    `json:"tx_hash"`
	BlockNumber      int64     `json:"block_number"`
	FromAddress      string    `json:"from_address"`
	ToAddress        string    `json:"to_address"`
	Value            string    `json:"value"`
	GasUsed          int64     `json:"gas_used"`
	FunctionSelector *string   `json:"function_selector,omitempty"`
	FunctionName     *string   `json:"function_name,omitempty"`
	Status           int16     `json:"status"`
	Timestamp        time.Time `json:"timestamp"`
}

// FuncSig 函数签名
type FuncSig struct {
	Selector     string `json:"selector"`
	Signature    string `json:"signature"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	IsPrivileged bool   `json:"is_privileged"`
}
