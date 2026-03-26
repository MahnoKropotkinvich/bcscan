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

	// ====== 跨交易上下文数据 ======

	// 地址行为画像
	SenderProfile *AddressProfileData // 发送方画像

	// 权限检查结果
	PrivilegeCheck *PrivilegeCheckData // 权限检查

	// 提取的数据（从 Extract 规则中提取）
	ExtractedData map[string]interface{}
}

// AddressProfileData 地址画像数据（嵌入 EvaluationContext）
type AddressProfileData struct {
	RecentTxCount         int     // 最近窗口内的交易数
	RecentTargetContracts int     // 最近窗口内交互的不同合约数
	RecentTotalValue      float64 // 最近窗口内的总转账金额 (ETH)
	RecentFailedTxCount   int     // 最近窗口内失败交易数
	RecentCallsToContract int     // 最近窗口内对当前目标合约的调用数
	TotalTxCount          int64   // 累计交易数
	IsPrivileged          bool    // 是否特权地址
}

// PrivilegeCheckData 权限检查结果（嵌入 EvaluationContext）
type PrivilegeCheckData struct {
	IsPrivilegedCall  bool   // 是否特权调用
	CallerAuthorized  bool   // 调用者是否被授权
	PrivilegeLevel    string // 特权级别: critical/high/medium/low
	RequiredRole      string // 需要的角色
	HasDelegatecall   bool   // 调用链中是否有 delegatecall
	CallerIsContract  bool   // 调用者是否为合约
	IntermediaryCount int    // 中间代理合约数量
	FunctionName      string // 特权函数名
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
