package ruleengine

import (
	"github.com/haswell/bcscan/internal/models"
)

// EvaluationContext 求值上下文，包含规则执行所需的所有数据
type EvaluationContext struct {
	// 基础数据
	Transaction *models.Transaction
	Block       *models.Block
	Events      []*models.Event

	// 运行时数据
	CallDepth    int               // 调用深度
	CallCount    int               // 调用次数
	CallTrace    []string          // 调用轨迹
	StateChanges map[string]string // 状态变化
	GasUsed      uint64            // Gas 使用量
	GasLimit     uint64            // Gas 限制

	// 提取的数据（从 Extract 规则中提取）
	ExtractedData map[string]interface{}
}

// NewEvaluationContext 创建新的求值上下文
func NewEvaluationContext(tx *models.Transaction, block *models.Block) *EvaluationContext {
	return &EvaluationContext{
		Transaction:   tx,
		Block:         block,
		Events:        make([]*models.Event, 0),
		CallTrace:     make([]string, 0),
		StateChanges:  make(map[string]string),
		ExtractedData: make(map[string]interface{}),
	}
}

// SetExtractedValue 设置提取的值
func (ctx *EvaluationContext) SetExtractedValue(key string, value interface{}) {
	ctx.ExtractedData[key] = value
}

// GetExtractedValue 获取提取的值
func (ctx *EvaluationContext) GetExtractedValue(key string) (interface{}, bool) {
	val, ok := ctx.ExtractedData[key]
	return val, ok
}
