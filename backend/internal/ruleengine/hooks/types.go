package hooks

import (
	"github.com/haswell/bcscan/internal/ruleengine"
)

// Hook 钩子接口
type Hook interface {
	Name() string
	Match(txData *TransactionData) bool
	Execute(ctx *ruleengine.EvaluationContext, rules []*ruleengine.Rule) ([]*RiskEvent, error)
}

// TransactionData 交易数据（从 Kafka 接收）
type TransactionData struct {
	TxHash           string      `json:"tx_hash"`
	BlockNumber      uint64      `json:"block_number"`
	FromAddress      string      `json:"from_address"`
	ToAddress        string      `json:"to_address"`
	Value            string      `json:"value"`
	GasPrice         uint64      `json:"gas_price"`
	GasUsed          uint64      `json:"gas_used"`
	GasLimit         uint64      `json:"gas_limit"`
	Status           uint64      `json:"status"`
	Timestamp        uint64      `json:"timestamp"`
	FunctionSelector string      `json:"function_selector"`
	InputData        string      `json:"input_data"`
	CallStack        []CallFrame `json:"call_stack"`
	Events           []EventLog  `json:"events"`
}

type CallFrame struct {
	Type     string `json:"type"`
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	Gas      uint64 `json:"gas"`
	GasUsed  uint64 `json:"gas_used"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	Depth    int    `json:"depth"`
	Function string `json:"function"`
}

type EventLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// RiskEvent 风险事件
type RiskEvent struct {
	RuleID      string
	RuleName    string
	Severity    string
	Score       int
	TxHash      string
	BlockNumber uint64
	Description string
	Metadata    map[string]interface{}
}
