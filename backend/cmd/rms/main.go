package main

import (
	"context"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/haswell/bcscan/internal/kafka"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Runtime Monitoring Service starting...")

	cfg := loadConfig()

	client, err := ethclient.Dial(cfg.EthNodeURL)
	if err != nil {
		logger.Fatal("Failed to connect to Ethereum node", zap.Error(err))
	}
	defer client.Close()

	chainID, _ := client.ChainID(context.Background())
	logger.Info("Connected to Ethereum network", zap.String("url", cfg.EthNodeURL), zap.String("chain_id", chainID.String()))

	producer := kafka.NewProducer([]string{cfg.KafkaBroker}, cfg.KafkaTopic, logger)
	defer producer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitorBlocks(ctx, client, producer, logger)

	waitForShutdown(logger)
}

type Config struct {
	EthNodeURL  string
	KafkaBroker string
	KafkaTopic  string
}

func loadConfig() *Config {
	return &Config{
		EthNodeURL:  getEnv("ETH_NODE_URL", "ws://ganache:8545"),
		KafkaBroker: getEnv("KAFKA_BROKER", "redpanda:9092"),
		KafkaTopic:  getEnv("KAFKA_TOPIC", "blockchain.transactions"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func waitForShutdown(logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	logger.Info("Shutdown signal received, stopping service...")
}

func monitorBlocks(ctx context.Context, client *ethclient.Client, producer *kafka.Producer, logger *zap.Logger) {
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		logger.Fatal("Failed to subscribe to new blocks", zap.Error(err))
	}
	defer sub.Unsubscribe()

	logger.Info("Monitoring blocks...")

	for {
		select {
		case err := <-sub.Err():
			logger.Error("Subscription error", zap.Error(err))
			return
		case header := <-headers:
			processBlock(ctx, client, producer, header.Number, logger)
		case <-ctx.Done():
			return
		}
	}
}

func processBlock(ctx context.Context, client *ethclient.Client, producer *kafka.Producer, blockNumber *big.Int, logger *zap.Logger) {
	block, err := client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		logger.Error("Failed to get block", zap.Error(err))
		return
	}

	logger.Info("Processing block", zap.Uint64("number", block.NumberU64()), zap.Int("txs", len(block.Transactions())))

	for _, tx := range block.Transactions() {
		processTransaction(ctx, client, producer, tx, block, logger)
	}
}

func processTransaction(ctx context.Context, client *ethclient.Client, producer *kafka.Producer, tx *types.Transaction, block *types.Block, logger *zap.Logger) {
	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		logger.Error("Failed to get receipt", zap.Error(err))
		return
	}

	// 构建完整的交易数据（包含调用栈）
	txData, err := buildTransactionData(ctx, client, tx, receipt, block)
	if err != nil {
		logger.Error("Failed to build transaction data", zap.Error(err))
		return
	}

	// 发送到 Kafka
	if err := producer.SendMessage(ctx, txData.TxHash, txData); err != nil {
		logger.Error("Failed to send transaction", zap.Error(err))
		return
	}

	logger.Info("Transaction processed",
		zap.String("tx_hash", txData.TxHash),
		zap.Int("call_stack_depth", len(txData.CallStack)),
		zap.Int("events", len(txData.Events)))
}
