package hooks

import (
	"github.com/haswell/bcscan/internal/ruleengine"
)

// Hook 钩子接口
type Hook interface {
	Name() string
	Execute(ctx *ruleengine.EvaluationContext, rules []*ruleengine.Rule) ([]*DetectionResult, error)
}

// DetectionResult Hook 检测结果（原 RiskEvent，重命名避免与 models.RiskEvent 混淆）
type DetectionResult struct {
	RuleID      string
	RuleName    string
	Severity    string
	Score       int
	TxHash      string
	BlockNumber uint64
	Description string
	Metadata    map[string]interface{}
}
