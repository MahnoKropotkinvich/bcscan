package ruleengine

import (
	"fmt"
	"strconv"
	"strings"
)

// Evaluator 表达式求值器
type Evaluator struct {
	// 可以添加缓存等优化
}

// NewEvaluator 创建新的求值器
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate 求值表达式，返回布尔结果
func (e *Evaluator) Evaluate(expression string, ctx *EvaluationContext) (bool, error) {
	if expression == "" {
		return true, nil
	}

	// 处理逻辑运算符 AND/OR
	if strings.Contains(expression, " AND ") {
		return e.evaluateAND(expression, ctx)
	}
	if strings.Contains(expression, " OR ") {
		return e.evaluateOR(expression, ctx)
	}

	// 单个条件求值
	return e.evaluateSingle(expression, ctx)
}

func (e *Evaluator) evaluateAND(expression string, ctx *EvaluationContext) (bool, error) {
	parts := strings.Split(expression, " AND ")
	for _, part := range parts {
		result, err := e.Evaluate(strings.TrimSpace(part), ctx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

func (e *Evaluator) evaluateOR(expression string, ctx *EvaluationContext) (bool, error) {
	parts := strings.Split(expression, " OR ")
	for _, part := range parts {
		result, err := e.Evaluate(strings.TrimSpace(part), ctx)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}
	return false, nil
}

func (e *Evaluator) evaluateSingle(expression string, ctx *EvaluationContext) (bool, error) {
	expression = strings.TrimSpace(expression)

	operators := []string{">=", "<=", "==", "!=", ">", "<"}

	for _, op := range operators {
		if strings.Contains(expression, op) {
			parts := strings.SplitN(expression, op, 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("invalid expression: %s", expression)
			}

			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			leftValue, err := e.getValue(left, ctx)
			if err != nil {
				return false, fmt.Errorf("failed to get value for '%s': %w", left, err)
			}

			rightValue, err := e.parseValue(right, ctx)
			if err != nil {
				return false, fmt.Errorf("failed to parse value '%s': %w", right, err)
			}

			return e.compare(leftValue, op, rightValue)
		}
	}

	return false, fmt.Errorf("no valid operator found in expression: %s", expression)
}

// getValue 从上下文中获取变量的值
func (e *Evaluator) getValue(varName string, ctx *EvaluationContext) (interface{}, error) {
	if val, ok := ctx.GetExtractedValue(varName); ok {
		return val, nil
	}

	switch varName {
	case "call_depth":
		return ctx.CallDepth, nil
	case "call_count":
		return ctx.CallCount, nil
	case "gas_used":
		return int(ctx.GasUsed), nil
	case "gas_limit":
		return int(ctx.GasLimit), nil
	case "reentrancy_detected":
		if val, ok := ctx.ExtractedData["reentrancy_detected"]; ok {
			return val, nil
		}
		return false, nil
	case "status":
		if ctx.Transaction != nil {
			return int(ctx.Transaction.Status), nil
		}
	case "value":
		if ctx.Transaction != nil {
			// 用 float64 解析 wei 值（int64 装不下 > 9.2 ETH 的 wei 值）
			val, err := strconv.ParseFloat(ctx.Transaction.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse value: %w", err)
			}
			return val, nil
		}
	case "gas_price":
		if ctx.Transaction != nil {
			return ctx.Transaction.GasPrice, nil
		}
	case "block_number":
		if ctx.Block != nil {
			return ctx.Block.BlockNumber, nil
		}

	// ====== 地址画像变量 ======
	case "sender_recent_tx_count":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.RecentTxCount, nil
		}
		return 0, nil
	case "sender_recent_contracts":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.RecentTargetContracts, nil
		}
		return 0, nil
	case "sender_recent_value":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.RecentTotalValue, nil
		}
		return 0.0, nil
	case "sender_recent_failed_tx":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.RecentFailedTxCount, nil
		}
		return 0, nil
	case "sender_recent_calls_to_target":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.RecentCallsToContract, nil
		}
		return 0, nil
	case "sender_total_tx":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.TotalTxCount, nil
		}
		return int64(0), nil
	case "sender_is_privileged":
		if ctx.SenderProfile != nil {
			return ctx.SenderProfile.IsPrivileged, nil
		}
		return false, nil

	// ====== 权限检查变量 ======
	case "is_privileged_call":
		if ctx.PrivilegeCheck != nil {
			return ctx.PrivilegeCheck.IsPrivilegedCall, nil
		}
		return false, nil
	case "caller_authorized":
		if ctx.PrivilegeCheck != nil {
			return ctx.PrivilegeCheck.CallerAuthorized, nil
		}
		return true, nil // 默认有权限（没有匹配权限条目时不算越权）
	case "privilege_level":
		if ctx.PrivilegeCheck != nil {
			return ctx.PrivilegeCheck.PrivilegeLevel, nil
		}
		return "", nil
	case "has_delegatecall":
		if ctx.PrivilegeCheck != nil {
			return ctx.PrivilegeCheck.HasDelegatecall, nil
		}
		return false, nil
	case "caller_is_contract":
		if ctx.PrivilegeCheck != nil {
			return ctx.PrivilegeCheck.CallerIsContract, nil
		}
		return false, nil
	case "intermediary_count":
		if ctx.PrivilegeCheck != nil {
			return ctx.PrivilegeCheck.IntermediaryCount, nil
		}
		return 0, nil
	}

	return nil, fmt.Errorf("unknown variable: %s", varName)
}

// parseValue 解析值（可能是常量或变量）
func (e *Evaluator) parseValue(value string, ctx *EvaluationContext) (interface{}, error) {
	// 尝试解析为整数（不溢出时用 int64）
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal, nil
	}

	// int64 溢出或科学计数法 → 用 float64
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}

	// 尝试解析为布尔值
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal, nil
	}

	// 去除引号（字符串字面量）
	if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
		return value[1 : len(value)-1], nil
	}

	// 当作变量名
	return e.getValue(value, ctx)
}

// compare 比较两个值
func (e *Evaluator) compare(left interface{}, operator string, right interface{}) (bool, error) {
	// 都能转 float64 就用 float64 比较（覆盖 int 和大数）
	leftFloat, leftOk := toFloat64(left)
	rightFloat, rightOk := toFloat64(right)

	if leftOk && rightOk {
		return compareFloat64(leftFloat, operator, rightFloat)
	}

	// 字符串比较
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch operator {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	default:
		return false, fmt.Errorf("unsupported operator '%s' for string comparison", operator)
	}
}

// toFloat64 尝试将值转换为 float64
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

// compareFloat64 比较两个 float64 值
func compareFloat64(left float64, operator string, right float64) (bool, error) {
	switch operator {
	case ">":
		return left > right, nil
	case "<":
		return left < right, nil
	case ">=":
		return left >= right, nil
	case "<=":
		return left <= right, nil
	case "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}
