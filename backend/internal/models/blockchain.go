package models

// TransactionData 完整的交易数据（Kafka 消息格式）
// 统一用于 RMS 发送和 RDS 接收
type TransactionData struct {
	// 基础信息
	TxHash      string `json:"tx_hash"`
	BlockNumber uint64 `json:"block_number"`
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Value       string `json:"value"`
	GasPrice    uint64 `json:"gas_price"`
	GasUsed     uint64 `json:"gas_used"`
	GasLimit    uint64 `json:"gas_limit"`
	Status      uint64 `json:"status"`
	Timestamp   uint64 `json:"timestamp"`

	// 函数调用信息
	FunctionSelector string `json:"function_selector"` // 前 4 字节
	InputData        string `json:"input_data"`

	// 调用栈
	CallStack []CallFrame `json:"call_stack"`

	// 事件日志
	Events []EventLog `json:"events"`
}

// CallFrame 调用帧
type CallFrame struct {
	Type     string `json:"type"`     // CALL, DELEGATECALL, STATICCALL, CREATE
	From     string `json:"from"`     // 调用者
	To       string `json:"to"`       // 被调用者
	Value    string `json:"value"`    // 转账金额
	Gas      uint64 `json:"gas"`      // Gas
	GasUsed  uint64 `json:"gas_used"` // 实际使用的 Gas
	Input    string `json:"input"`    // 输入数据
	Output   string `json:"output"`   // 输出数据
	Error    string `json:"error"`    // 错误信息
	Depth    int    `json:"depth"`    // 调用深度
	Function string `json:"function"` // 函数签名（如果能解析）
}

// EventLog 事件日志
type EventLog struct {
	Address string   `json:"address"` // 合约地址
	Topics  []string `json:"topics"`  // 事件主题
	Data    string   `json:"data"`    // 事件数据
}

// DetectReentrancyPattern 检测重入攻击模式（A->B->A 调用链）
func DetectReentrancyPattern(callStack []CallFrame) bool {
	addressCalls := make(map[string][]int)

	for i, frame := range callStack {
		if frame.To == "" {
			continue
		}
		addressCalls[frame.To] = append(addressCalls[frame.To], i)
	}

	// 检查是否有地址被多次调用且中间插入了其他调用
	for _, indices := range addressCalls {
		if len(indices) >= 2 {
			for i := 0; i < len(indices)-1; i++ {
				if indices[i+1] > indices[i]+1 {
					return true
				}
			}
		}
	}

	return false
}

// CalculateMaxCallDepth 计算最大调用深度
func CalculateMaxCallDepth(callStack []CallFrame) int {
	maxDepth := 0
	for _, frame := range callStack {
		if frame.Depth > maxDepth {
			maxDepth = frame.Depth
		}
	}
	return maxDepth
}

// ContainsFunctionCall 检查调用栈是否包含特定函数选择器
func ContainsFunctionCall(callStack []CallFrame, functionSelector string) bool {
	for _, frame := range callStack {
		if len(frame.Input) >= len(functionSelector) && frame.Input[:len(functionSelector)] == functionSelector {
			return true
		}
	}
	return false
}
