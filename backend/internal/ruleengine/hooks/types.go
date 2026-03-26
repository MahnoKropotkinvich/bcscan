package hooks

import (
	"github.com/haswell/bcscan/internal/ruleengine"
	"github.com/haswell/bcscan/internal/types"
)

// Hook 钩子接口
type Hook interface {
	Name() string
	Match(txData *types.TransactionData) bool
	Execute(ctx *ruleengine.EvaluationContext, rules []*ruleengine.Rule) ([]*RiskEvent, error)
}

// RiskEvent 风险事件（Hook 检测结果）
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
