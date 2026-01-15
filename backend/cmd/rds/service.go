package main

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/kafka"
	"github.com/haswell/bcscan/internal/models"
	"github.com/haswell/bcscan/internal/repository"
	"github.com/haswell/bcscan/internal/ruleengine"
	"github.com/haswell/bcscan/internal/ruleengine/hooks"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// RDSService 风险检测服务
type RDSService struct {
	db            *sql.DB
	cfg           *Config
	logger        *zap.Logger
	kafkaConsumer *kafka.Consumer
	hookManager   *hooks.Manager
	ruleManager   *ruleengine.RuleManager
	scorer        *ruleengine.Scorer
	executor      *ruleengine.Executor
	running       bool
}

// NewRDSService 创建新的 RDS 服务
func NewRDSService(db *sql.DB, cfg *Config, logger *zap.Logger) *RDSService {
	redis := cache.NewRedisClient(cfg.RedisAddr)
	repo := repository.NewRiskEventRepository(db, redis, logger)
	return &RDSService{
		db:          db,
		cfg:         cfg,
		logger:      logger,
		hookManager: hooks.NewManager(),
		ruleManager: ruleengine.NewRuleManager(cfg.RulesPath, redis, logger),
		scorer:      ruleengine.NewScorer(),
		executor:    ruleengine.NewExecutor(repo),
		running:     false,
	}
}

// Start 启动服务
func (s *RDSService) Start() error {
	s.logger.Info("Initializing service...")

	// 1. 加载规则
	if err := s.loadRules(); err != nil {
		return err
	}

	// 2. 注册钩子
	s.registerHooks()

	// 3. 初始化 Kafka 消费者
	s.kafkaConsumer = kafka.NewConsumer(
		[]string{s.cfg.KafkaBroker},
		s.cfg.KafkaTopic,
		"rds-consumer-group",
		s.logger,
	)

	// 4. 启动规则热加载
	go s.ruleManager.SubscribeUpdates(context.Background())

	// 5. 启动消息处理
	go s.processMessages()

	// 6. 标记为运行中
	s.running = true

	s.logger.Info("Service started successfully", zap.Int("rules", len(s.ruleManager.GetRules())))
	return nil
}

// Stop 停止服务
func (s *RDSService) Stop() {
	s.running = false
	s.logger.Info("Service stopped")
}

// loadRules 加载规则
func (s *RDSService) loadRules() error {
	s.logger.Info("Loading rules", zap.String("path", s.cfg.RulesPath))

	ctx := context.Background()
	if err := s.ruleManager.LoadRules(ctx); err != nil {
		return err
	}

	s.logger.Info("Rules loaded", zap.Int("enabled_rules", len(s.ruleManager.GetRules())))
	return nil
}

// registerHooks 注册钩子
func (s *RDSService) registerHooks() {
	// 注册合约函数调用钩子
	contractFunctionHook := hooks.NewContractFunctionHook()
	s.hookManager.Register(contractFunctionHook)

	s.logger.Info("Registered hooks", zap.String("hooks", "contract_function_call"))
}

// processMessages 处理 Kafka 消息
func (s *RDSService) processMessages() {
	ctx := context.Background()

	s.logger.Info("Starting message processing...")

	for s.running {
		msg, err := s.kafkaConsumer.ReadMessage(ctx)
		if err != nil {
			s.logger.Error("Failed to read message", zap.Error(err))
			continue
		}

		// 处理交易消息
		if err := s.processTransaction(&msg); err != nil {
			s.logger.Error("Failed to process transaction", zap.Error(err))
		}
	}
}

// processTransaction 处理单个交易
func (s *RDSService) processTransaction(msg *kafkago.Message) error {
	// 1. 解析消息为交易数据（新格式，包含调用栈）
	var txData hooks.TransactionData
	if err := json.Unmarshal(msg.Value, &txData); err != nil {
		return err
	}

	// 2. 转换为 models.Transaction（用于兼容）
	tx := &models.Transaction{
		TxHash:      txData.TxHash,
		BlockNumber: int64(txData.BlockNumber),
		FromAddress: txData.FromAddress,
		ToAddress:   txData.ToAddress,
		Value:       txData.Value,
		GasPrice:    int64(txData.GasPrice),
		GasUsed:     int64(txData.GasUsed),
		Status:      int16(txData.Status),
	}

	// 3. 创建评估上下文并填充运行时数据
	ctx := ruleengine.NewEvaluationContext(tx, nil)
	ctx.CallDepth = hooks.CalculateMaxCallDepth(txData.CallStack)
	ctx.CallCount = len(txData.CallStack)
	ctx.GasUsed = txData.GasUsed
	ctx.GasLimit = txData.GasLimit

	// 填充调用轨迹
	for _, frame := range txData.CallStack {
		ctx.CallTrace = append(ctx.CallTrace, frame.To)
	}

	// 检测重入模式
	if hooks.DetectReentrancyPattern(txData.CallStack) {
		ctx.ExtractedData["reentrancy_detected"] = true
	}

	// 4. 触发 hook
	rules := s.ruleManager.GetRules()
	events, err := s.hookManager.Trigger("contract_function_call", ctx, rules)
	if err != nil {
		return err
	}

	// 5. 处理风险事件
	for _, event := range events {
		var matchedRule *ruleengine.Rule
		for _, rule := range rules {
			if rule.Metadata.Name == event.RuleID {
				matchedRule = rule
				break
			}
		}

		if matchedRule == nil {
			continue
		}

		score, err := s.scorer.CalculateScore(matchedRule, ctx)
		if err != nil {
			s.logger.Error("Failed to calculate score", zap.Error(err))
			continue
		}

		event.Score = score

		if err := s.executor.Execute(matchedRule, ctx, score); err != nil {
			s.logger.Error("Failed to execute actions", zap.Error(err))
		}

		s.logger.Info("Risk event detected",
			zap.String("rule", event.RuleName),
			zap.String("tx_hash", event.TxHash),
			zap.Int("score", score),
			zap.Int("call_depth", ctx.CallDepth))
	}

	return nil
}
