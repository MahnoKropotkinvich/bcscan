# Hook 系统重新设计

## 当前问题
- `post_transaction` 太泛化，无法针对特定合约
- 缺少调用栈信息
- 无法监听特定函数

## 正确设计

### Hook 类型

1. **contract_function_call** - 监听合约函数调用
   ```yaml
   hooks:
     - type: contract_function_call
       contract: "0x123..."
       function: "withdraw(uint256)"
       analyze_call_stack: true
   ```

2. **contract_event** - 监听合约事件
   ```yaml
   hooks:
     - type: contract_event
       contract: "0x123..."
       event: "Transfer(address,address,uint256)"
   ```

3. **call_pattern** - 监听调用模式
   ```yaml
   hooks:
     - type: call_pattern
       pattern: "A->B->A"  # 重入模式
   ```

## 实现需求

### RMS 需要：
1. 解析交易 input data (函数签名)
2. 使用 `debug_traceTransaction` 获取调用栈
3. 解析 logs 获取事件
4. 发送完整的调用信息到 Kafka

### 消息格式：
```json
{
  "tx_hash": "0x...",
  "from": "0x...",
  "to": "0x...",
  "value": "1000",
  "function": "withdraw(uint256)",
  "function_selector": "0x2e1a7d4d",
  "call_stack": [
    {
      "from": "0xA",
      "to": "0xB",
      "function": "withdraw",
      "depth": 0
    },
    {
      "from": "0xB",
      "to": "0xC",
      "function": "transfer",
      "depth": 1
    },
    {
      "from": "0xC",
      "to": "0xB",
      "function": "callback",
      "depth": 2
    }
  ],
  "events": [
    {
      "contract": "0xB",
      "event": "Transfer",
      "data": {...}
    }
  ],
  "state_changes": {
    "0xB:balance": "1000->0"
  }
}
```

### RDS Hook 系统：
```go
type Hook interface {
    Name() string
    Match(tx *TransactionData) bool  // 是否匹配此 hook
    Execute(ctx *EvaluationContext, rules []*Rule) ([]*RiskEvent, error)
}

// 合约函数调用 Hook
type ContractFunctionHook struct {
    ContractAddress string
    FunctionSelector string
}

func (h *ContractFunctionHook) Match(tx *TransactionData) bool {
    return tx.To == h.ContractAddress && 
           tx.FunctionSelector == h.FunctionSelector
}
```

## 规则示例

### 重入攻击检测
```yaml
metadata:
  name: "reentrancy-attack"

hooks:
  - type: contract_function_call
    contract: "0x..." # 目标合约
    function: "withdraw(uint256)"
    analyze_call_stack: true

triggers:
  conditions:
    - type: call_pattern
      pattern: "A->B->A"  # 检测重入模式
      description: "同一合约被重复调用"
    
    - type: call_depth
      operator: ">"
      value: 2

actions:
  - type: alert
    severity: critical
```

### 闪电贷攻击检测
```yaml
metadata:
  name: "flash-loan-attack"

hooks:
  - type: contract_event
    event: "FlashLoan(address,uint256)"

triggers:
  conditions:
    - type: large_value_transfer
      operator: ">"
      value: "1000000000000000000000"  # > 1000 ETH
    
    - type: same_block_repay
      check: true

actions:
  - type: alert
```

## 下一步

需要实现：
1. RMS 增强：获取调用栈和事件
2. 重新设计 Hook 接口
3. 更新规则格式
4. 实现模式匹配引擎
