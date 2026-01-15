package ruleengine

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// Executor 动作执行器
type Executor struct {
	db *sql.DB
}

// NewExecutor 创建新的执行器
func NewExecutor(db *sql.DB) *Executor {
	return &Executor{
		db: db,
	}
}

// Execute 执行规则动作
func (e *Executor) Execute(rule *Rule, ctx *EvaluationContext, score int) error {
	for _, action := range rule.Actions {
		if err := e.executeAction(action, rule, ctx, score); err != nil {
			log.Printf("[ERROR] Failed to execute action %s: %v", action.Type, err)
			// 继续执行其他动作，不中断
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
		log.Printf("[WARN] Unknown action type: %s", action.Type)
		return nil
	}
}

// executeAlert 执行告警动作
func (e *Executor) executeAlert(action RuleAction, rule *Rule, ctx *EvaluationContext, score int) error {
	// 替换消息模板中的变量
	message := e.replaceVariables(action.Message, ctx)
	title := e.replaceVariables(action.Title, ctx)

	log.Printf("[ALERT] %s - %s (Score: %d)", title, message, score)

	// TODO: 发送到告警服务
	return nil
}

// logRiskEvent 记录风险事件到数据库
func (e *Executor) logRiskEvent(action RuleAction, rule *Rule, ctx *EvaluationContext, score int) error {
	if e.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	var txHash string
	var contractAddr string

	if ctx.Transaction != nil {
		txHash = ctx.Transaction.TxHash
		contractAddr = ctx.Transaction.ToAddress
	}

	query := `
		INSERT INTO risk_events (event_type, severity, contract_address, tx_hash, description, score, detected_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err := e.db.Exec(query,
		rule.Metadata.Name,
		rule.Config.Severity,
		contractAddr,
		txHash,
		rule.Metadata.Description,
		score,
	)

	return err
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
