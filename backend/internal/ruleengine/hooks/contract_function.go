package hooks

import (
	"fmt"

	"github.com/haswell/bcscan/internal/ruleengine"
	"github.com/haswell/bcscan/internal/types"
)

// ContractFunctionHook 合约函数调用钩子
type ContractFunctionHook struct {
	evaluator *ruleengine.Evaluator
}

func NewContractFunctionHook() *ContractFunctionHook {
	return &ContractFunctionHook{
		evaluator: ruleengine.NewEvaluator(),
	}
}

func (h *ContractFunctionHook) Name() string {
	return "contract_function_call"
}

func (h *ContractFunctionHook) Match(txData *types.TransactionData) bool {
	// 匹配所有交易（包括纯 ETH 转账），让规则条件来过滤
	return true
}

func (h *ContractFunctionHook) Execute(ctx *ruleengine.EvaluationContext, rules []*ruleengine.Rule) ([]*RiskEvent, error) {
	var events []*RiskEvent

	for _, rule := range rules {
		if !rule.Metadata.Enabled {
			continue
		}

		// 检查规则是否包含此 hook
		if !h.hasHook(rule, "contract_function_call") {
			continue
		}

		// 评估规则
		matched, err := h.evaluateRule(rule, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule %s: %w", rule.Metadata.Name, err)
		}

		if matched {
			event := h.createRiskEvent(rule, ctx)
			events = append(events, event)
		}
	}

	return events, nil
}

func (h *ContractFunctionHook) hasHook(rule *ruleengine.Rule, hookName string) bool {
	for _, hook := range rule.Config.Hooks {
		if hook == hookName {
			return true
		}
	}
	return false
}

func (h *ContractFunctionHook) evaluateRule(rule *ruleengine.Rule, ctx *ruleengine.EvaluationContext) (bool, error) {
	if len(rule.Triggers.Conditions) == 0 {
		return true, nil
	}

	operator := rule.Triggers.Operator
	if operator == "" {
		operator = "AND"
	}

	results := make([]bool, 0, len(rule.Triggers.Conditions))

	for _, condition := range rule.Triggers.Conditions {
		result, err := h.evaluateCondition(condition, ctx)
		if err != nil {
			return false, err
		}
		results = append(results, result)
	}

	switch operator {
	case "AND":
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil
	case "OR":
		for _, r := range results {
			if r {
				return true, nil
			}
		}
		return false, nil
	}

	return false, fmt.Errorf("unsupported operator: %s", operator)
}

func (h *ContractFunctionHook) evaluateCondition(condition ruleengine.RuleCondition, ctx *ruleengine.EvaluationContext) (bool, error) {
	expression := fmt.Sprintf("%s %s %v", condition.Type, condition.Operator, condition.Value)
	return h.evaluator.Evaluate(expression, ctx)
}

func (h *ContractFunctionHook) createRiskEvent(rule *ruleengine.Rule, ctx *ruleengine.EvaluationContext) *RiskEvent {
	event := &RiskEvent{
		RuleID:   rule.Metadata.Name,
		RuleName: rule.Metadata.Name,
		Severity: rule.Config.Severity,
		Score:    rule.Scoring.BaseScore,
		Metadata: make(map[string]interface{}),
	}

	if ctx.Transaction != nil {
		event.TxHash = ctx.Transaction.TxHash
		event.BlockNumber = uint64(ctx.Transaction.BlockNumber)
	}

	for key, value := range ctx.ExtractedData {
		event.Metadata[key] = value
	}

	return event
}
