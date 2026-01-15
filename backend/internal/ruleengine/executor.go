package ruleengine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/haswell/bcscan/internal/models"
	"github.com/haswell/bcscan/internal/repository"
	"go.uber.org/zap"
)

// Executor 动作执行器
type Executor struct {
	repo   *repository.RiskEventRepository
	logger *zap.Logger
}

// NewExecutor 创建新的执行器
func NewExecutor(repo *repository.RiskEventRepository) *Executor {
	logger, _ := zap.NewProduction()
	return &Executor{
		repo:   repo,
		logger: logger,
	}
}

// Execute 执行规则动作
func (e *Executor) Execute(rule *Rule, ctx *EvaluationContext, score int) error {
	for _, action := range rule.Actions {
		if err := e.executeAction(action, rule, ctx, score); err != nil {
			e.logger.Error("Failed to execute action",
				zap.String("action", action.Type),
				zap.Error(err))
		}
	}
	return nil
}

// executeAction 执行单个动作
func (e *Executor) executeAction(action RuleAction, rule *Rule, ctx *EvaluationContext, score int) error {
	switch action.Type {
	case "alert":
		return e.executeAlert(action, rule, ctx, score)
	case "log_risk_event":
		return e.logRiskEvent(action, rule, ctx, score)
	default:
		e.logger.Warn("Unknown action type", zap.String("type", action.Type))
		return nil
	}
}

// executeAlert 执行告警动作
func (e *Executor) executeAlert(action RuleAction, rule *Rule, ctx *EvaluationContext, score int) error {
	message := e.replaceVariables(action.Message, ctx)
	title := e.replaceVariables(action.Title, ctx)

	e.logger.Info("ALERT",
		zap.String("title", title),
		zap.String("message", message),
		zap.Int("score", score))

	return nil
}

// logRiskEvent 记录风险事件到数据库
func (e *Executor) logRiskEvent(action RuleAction, rule *Rule, ctx *EvaluationContext, score int) error {
	if e.repo == nil {
		return fmt.Errorf("repository is nil")
	}

	var txHash string
	var contractAddr string

	if ctx.Transaction != nil {
		txHash = ctx.Transaction.TxHash
		contractAddr = ctx.Transaction.ToAddress
	}

	event := &models.RiskEvent{
		EventType:       rule.Metadata.Name,
		Severity:        rule.Config.Severity,
		ContractAddress: contractAddr,
		TxHash:          txHash,
		Description:     rule.Metadata.Description,
		Score:           score,
		DetectedAt:      time.Now(),
	}

	return e.repo.Create(context.Background(), event)
}

// replaceVariables 替换消息模板中的变量
func (e *Executor) replaceVariables(template string, ctx *EvaluationContext) string {
	result := template

	// 替换提取的数据
	for key, value := range ctx.ExtractedData {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	// 替换交易信息
	if ctx.Transaction != nil {
		result = strings.ReplaceAll(result, "{{tx_hash}}", ctx.Transaction.TxHash)
		result = strings.ReplaceAll(result, "{{from_address}}", ctx.Transaction.FromAddress)
		result = strings.ReplaceAll(result, "{{to_address}}", ctx.Transaction.ToAddress)
	}

	// 替换运行时数据
	result = strings.ReplaceAll(result, "{{call_depth}}", fmt.Sprintf("%d", ctx.CallDepth))
	result = strings.ReplaceAll(result, "{{call_count}}", fmt.Sprintf("%d", ctx.CallCount))

	return result
}
