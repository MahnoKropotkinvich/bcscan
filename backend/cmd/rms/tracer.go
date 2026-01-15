package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TraceResult debug_traceTransaction 返回结果
type TraceResult struct {
	Type    string        `json:"type"`
	From    string        `json:"from"`
	To      string        `json:"to"`
	Value   string        `json:"value"`
	Gas     string        `json:"gas"`
	GasUsed string        `json:"gasUsed"`
	Input   string        `json:"input"`
	Output  string        `json:"output"`
	Error   string        `json:"error"`
	Calls   []TraceResult `json:"calls"`
}

func buildTransactionData(ctx context.Context, client *ethclient.Client, tx *types.Transaction, receipt *types.Receipt, block *types.Block) (*TransactionData, error) {
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	// 提取函数选择器
	functionSelector := ""
	inputData := ""
	if len(tx.Data()) >= 4 {
		functionSelector = "0x" + hex.EncodeToString(tx.Data()[:4])
		inputData = "0x" + hex.EncodeToString(tx.Data())
	}

	txData := &TransactionData{
		TxHash:           tx.Hash().Hex(),
		BlockNumber:      block.NumberU64(),
		FromAddress:      from.Hex(),
		ToAddress:        to,
		Value:            tx.Value().String(),
		GasPrice:         tx.GasPrice().Uint64(),
		GasUsed:          receipt.GasUsed,
		GasLimit:         tx.Gas(),
		Status:           receipt.Status,
		Timestamp:        block.Time(),
		FunctionSelector: functionSelector,
		InputData:        inputData,
		CallStack:        []CallFrame{},
		Events:           []EventLog{},
	}

	// 追踪调用栈（仅对合约调用）
	if to != "" && len(tx.Data()) > 0 {
		trace, err := traceTransaction(ctx, client, tx.Hash())
		if err == nil {
			txData.CallStack = parseCallStack(trace, 0)
		}
	}

	// 解析事件
	for _, log := range receipt.Logs {
		topics := make([]string, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = topic.Hex()
		}
		txData.Events = append(txData.Events, EventLog{
			Address: log.Address.Hex(),
			Topics:  topics,
			Data:    "0x" + hex.EncodeToString(log.Data),
		})
	}

	return txData, nil
}

func traceTransaction(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*TraceResult, error) {
	var result TraceResult

	// 调用 debug_traceTransaction
	err := client.Client().CallContext(ctx, &result, "debug_traceTransaction", txHash, map[string]interface{}{
		"tracer": "callTracer",
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func parseCallStack(trace *TraceResult, depth int) []CallFrame {
	if trace == nil {
		return []CallFrame{}
	}

	frames := []CallFrame{}

	// 当前调用帧
	frame := CallFrame{
		Type:     trace.Type,
		From:     trace.From,
		To:       trace.To,
		Value:    trace.Value,
		Gas:      parseHexUint64(trace.Gas),
		GasUsed:  parseHexUint64(trace.GasUsed),
		Input:    trace.Input,
		Output:   trace.Output,
		Error:    trace.Error,
		Depth:    depth,
		Function: extractFunctionSignature(trace.Input),
	}
	frames = append(frames, frame)

	// 递归处理子调用
	for _, call := range trace.Calls {
		frames = append(frames, parseCallStack(&call, depth+1)...)
	}

	return frames
}

func parseHexUint64(hexStr string) uint64 {
	if hexStr == "" || hexStr == "0x" {
		return 0
	}
	var result big.Int
	result.SetString(hexStr[2:], 16)
	return result.Uint64()
}

func extractFunctionSignature(input string) string {
	if len(input) < 10 {
		return ""
	}
	return input[:10] // 0x + 8 hex chars
}
