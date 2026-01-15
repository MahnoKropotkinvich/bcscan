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
// 支持的表达式示例:
//   - "call_depth > 3"
//   - "gas_used > 1000000"
//   - "call_depth > 3 AND gas_used > 1000000"
//   - "status == 1 OR value > 1000"
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

// evaluateAND 求值 AND 表达式
func (e *Evaluator) evaluateAND(expression string, ctx *EvaluationContext) (bool, error) {
	parts := strings.Split(expression, " AND ")
	for _, part := range parts {
		result, err := e.Evaluate(strings.TrimSpace(part), ctx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil // 短路求值
		}
	}
	return true, nil
}

// evaluateOR 求值 OR 表达式
func (e *Evaluator) evaluateOR(expression string, ctx *EvaluationContext) (bool, error) {
	parts := strings.Split(expression, " OR ")
	for _, part := range parts {
		result, err := e.Evaluate(strings.TrimSpace(part), ctx)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil // 短路求值
		}
	}
	return false, nil
}

// evaluateSingle 求值单个比较表达式
// 支持的运算符: >, <, >=, <=, ==, !=
func (e *Evaluator) evaluateSingle(expression string, ctx *EvaluationContext) (bool, error) {
	expression = strings.TrimSpace(expression)

	// 尝试匹配各种运算符
	operators := []string{">=", "<=", "==", "!=", ">", "<"}

	for _, op := range operators {
		if strings.Contains(expression, op) {
			parts := strings.SplitN(expression, op, 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("invalid expression: %s", expression)
			}

			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// 获取左侧变量的值
			leftValue, err := e.getValue(left, ctx)
			if err != nil {
				return false, fmt.Errorf("failed to get value for '%s': %w", left, err)
			}

			// 获取右侧值（可能是变量或常量）
			rightValue, err := e.parseValue(right, ctx)
			if err != nil {
				return false, fmt.Errorf("failed to parse value '%s': %w", right, err)
			}

			// 执行比较
			return e.compare(leftValue, op, rightValue)
		}
	}

	return false, fmt.Errorf("no valid operator found in expression: %s", expression)
}

// getValue 从上下文中获取变量的值
func (e *Evaluator) getValue(varName string, ctx *EvaluationContext) (interface{}, error) {
	// 先检查提取的数据
	if val, ok := ctx.GetExtractedValue(varName); ok {
		return val, nil
	}

	// 从上下文中获取预定义的字段
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
			// 将字符串转换为数字
			val, err := strconv.ParseInt(ctx.Transaction.Value, 10, 64)
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
	}

	return nil, fmt.Errorf("unknown variable: %s", varName)
}

// parseValue 解析值（可能是常量或变量）
func (e *Evaluator) parseValue(value string, ctx *EvaluationContext) (interface{}, error) {
	// 尝试解析为整数
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal, nil
	}

	// 尝试解析为浮点数
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}

	// 尝试解析为布尔值
	if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal, nil
	}

	// 去除引号（如果是字符串字面量）
	if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
		return value[1 : len(value)-1], nil
	}

	// 否则当作变量名处理
	return e.getValue(value, ctx)
}

// compare 比较两个值
func (e *Evaluator) compare(left interface{}, operator string, right interface{}) (bool, error) {
	// 尝试转换为 int64 进行比较
	leftInt, leftIsInt := toInt64(left)
	rightInt, rightIsInt := toInt64(right)

	if leftIsInt && rightIsInt {
		return compareInt64(leftInt, operator, rightInt)
	}

	// 尝试转换为 float64 进行比较
	leftFloat, leftIsFloat := toFloat64(left)
	rightFloat, rightIsFloat := toFloat64(right)

	if leftIsFloat && rightIsFloat {
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

// toInt64 尝试将值转换为 int64
func toInt64(val interface{}) (int64, bool) {
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	default:
		return 0, false
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
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

// compareInt64 比较两个 int64 值
func compareInt64(left int64, operator string, right int64) (bool, error) {
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
