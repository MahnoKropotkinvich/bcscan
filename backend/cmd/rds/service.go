package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/kafka"
	"github.com/haswell/bcscan/internal/models"
	"github.com/haswell/bcscan/internal/privilege"
	"github.com/haswell/bcscan/internal/repository"
	"github.com/haswell/bcscan/internal/ruleengine"
	"github.com/haswell/bcscan/internal/ruleengine/hooks"
	"github.com/haswell/bcscan/internal/tracker"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// RDSService 风险检测服务
type RDSService struct {
	db            *sql.DB
	cfg           *Config
	logger        *zap.Logger
	redis         *cache.RedisClient
	kafkaConsumer *kafka.Consumer
	hookManager   *hooks.Manager
	ruleManager   *ruleengine.RuleManager
	scorer        *ruleengine.Scorer
	executor      *ruleengine.Executor
	riskRepo      *repository.RiskEventRepository
	// 跨交易上下文分析
	tracker      *tracker.Tracker             // 地址行为追踪器
	privRegistry *privilege.PrivilegeRegistry // 权限注册表
	running      atomic.Bool
	cancel       context.CancelFunc
}

// NewRDSService 创建新的 RDS 服务
func NewRDSService(db *sql.DB, cfg *Config, logger *zap.Logger) *RDSService {
	redis := cache.NewRedisClient(cfg.RedisAddr)
	repo := repository.NewRiskEventRepository(db, redis, logger)
	return &RDSService{
		db:           db,
		cfg:          cfg,
		logger:       logger,
		redis:        redis,
		hookManager:  hooks.NewManager(),
		ruleManager:  ruleengine.NewRuleManager(cfg.RulesPath, redis, logger),
		scorer:       ruleengine.NewScorer(),
		executor:     ruleengine.NewExecutor(repo, logger),
		riskRepo:     repo,
		tracker:      tracker.NewTracker(cfg.RedisAddr, logger),
		privRegistry: privilege.NewPrivilegeRegistry(db, logger),
	}
}

// Start 启动服务
func (s *RDSService) Start() error {
	s.logger.Info("Initializing service...")

	// 1. 加载规则
	if err := s.loadRules(); err != nil {
		return err
	}

	// 2. 加载权限注册表
	ctx := context.Background()
	if err := s.privRegistry.Load(ctx); err != nil {
		s.logger.Warn("Failed to load privilege registry (will retry)", zap.Error(err))
	}

	// 3. 注册钩子
	s.registerHooks()

	// 4. 初始化 Kafka 消费者
	s.kafkaConsumer = kafka.NewConsumer(
		[]string{s.cfg.KafkaBroker},
		s.cfg.KafkaTopic,
		"rds-consumer-group",
		s.logger,
	)

	// 5. 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// 6. 启动规则热加载
	go s.ruleManager.SubscribeUpdates(ctx)

	// 7. 启动权限表定期重载（每 5 分钟）
	go s.privRegistry.PeriodicReload(ctx, 5*time.Minute)

	// 8. 启动消息处理
	s.running.Store(true)
	go s.processMessages(ctx)

	s.logger.Info("Service started successfully",
		zap.Int("rules", len(s.ruleManager.GetRules())),
		zap.String("features", "profile_tracking,privilege_checking"))
	return nil
}

// Stop 停止服务
func (s *RDSService) Stop() {
	s.logger.Info("Stopping service...")
	s.running.Store(false)

	// 取消上下文，通知所有 goroutine
	if s.cancel != nil {
		s.cancel()
	}

	// 关闭 Kafka 消费者
	if s.kafkaConsumer != nil {
		s.kafkaConsumer.Close()
	}

	// 关闭 RiskEvent 仓储的写入 worker
	if s.riskRepo != nil {
		s.riskRepo.Close()
	}

	// 关闭行为追踪器
	if s.tracker != nil {
		s.tracker.Close()
	}

	// 关闭 Redis
	if s.redis != nil {
		s.redis.Close()
	}

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
	contractFunctionHook := hooks.NewContractFunctionHook()
	s.hookManager.Register(contractFunctionHook)
	s.logger.Info("Registered hooks", zap.String("hooks", "contract_function_call"))
}

// processMessages 处理 Kafka 消息
func (s *RDSService) processMessages(ctx context.Context) {
	s.logger.Info("Starting message processing...")

	for s.running.Load() {
		msg, err := s.kafkaConsumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				s.logger.Info("Message processing stopped: context cancelled")
				return
			}
			s.logger.Error("Failed to read message", zap.Error(err))
			continue
		}

		if err := s.processTransaction(ctx, &msg); err != nil {
			s.logger.Error("Failed to process transaction", zap.Error(err))
		}
	}
}

// processTransaction 处理单个交易
func (s *RDSService) processTransaction(ctx context.Context, msg *kafkago.Message) error {
	// 1. 解析消息为交易数据（现在使用 models.TransactionData）
	var txData models.TransactionData
	if err := json.Unmarshal(msg.Value, &txData); err != nil {
		return err
	}

	// 2. 转换为 models.Transaction（用于评估上下文）
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
	evalCtx := ruleengine.NewEvaluationContext(tx, nil)
	evalCtx.CallDepth = models.CalculateMaxCallDepth(txData.CallStack)
	evalCtx.CallCount = len(txData.CallStack)
	evalCtx.GasUsed = txData.GasUsed
	evalCtx.GasLimit = txData.GasLimit

	// 填充调用轨迹
	for _, frame := range txData.CallStack {
		evalCtx.CallTrace = append(evalCtx.CallTrace, frame.To)
	}

	// 检测重入模式
	if models.DetectReentrancyPattern(txData.CallStack) {
		evalCtx.ExtractedData["reentrancy_detected"] = true
	}

	// ====== 4. 跨交易上下文分析 ======

	// 4a. 记录交互事件到行为追踪器
	hasDelegatecall := false
	for _, frame := range txData.CallStack {
		if strings.EqualFold(frame.Type, "DELEGATECALL") {
			hasDelegatecall = true
			break
		}
	}

	interactionEvent := &tracker.InteractionEvent{
		TxHash:           txData.TxHash,
		FromAddress:      txData.FromAddress,
		ToAddress:        txData.ToAddress,
		Value:            txData.Value,
		GasUsed:          txData.GasUsed,
		FunctionSelector: txData.FunctionSelector,
		CallDepth:        evalCtx.CallDepth,
		HasDelegatecall:  hasDelegatecall,
		Status:           txData.Status,
		Timestamp:        time.Now(),
	}
	if err := s.tracker.RecordInteraction(ctx, interactionEvent); err != nil {
		s.logger.Warn("Failed to record interaction", zap.Error(err))
	}

	// 4b. 获取发送方行为画像（直接赋值，消除平行结构）
	senderProfile, err := s.tracker.GetProfile(ctx, txData.FromAddress, txData.ToAddress)
	if err != nil {
		s.logger.Warn("Failed to get sender profile", zap.Error(err))
	}
	evalCtx.SenderProfile = senderProfile

	// 4c. 权限检查（直接使用 models.CallFrame，消除 CallFrameInfo 转换）
	privResult := s.privRegistry.CheckPrivilege(
		txData.FromAddress,
		txData.ToAddress,
		txData.FunctionSelector,
		txData.CallStack,
	)
	if privResult != nil {
		evalCtx.PrivilegeCheck = privResult
		if privResult.MatchedEntry != nil {
			evalCtx.ExtractedData["privilege_function"] = privResult.MatchedEntry.FunctionName
		}
		// 注入到 ExtractedData 供模板使用
		evalCtx.ExtractedData["privilege_level"] = privResult.PrivilegeLevel
		evalCtx.ExtractedData["required_role"] = privResult.RequiredRole
		evalCtx.ExtractedData["effective_caller"] = privResult.EffectiveCaller
		if len(privResult.IntermediaryContracts) > 0 {
			evalCtx.ExtractedData["intermediary_contracts"] = strings.Join(privResult.IntermediaryContracts, ", ")
		}
	}

	// 注入画像数据到 ExtractedData 供模板使用
	if senderProfile != nil {
		evalCtx.ExtractedData["sender_recent_tx_count"] = senderProfile.RecentTxCount
		evalCtx.ExtractedData["sender_recent_contracts"] = senderProfile.RecentTargetContracts
		evalCtx.ExtractedData["sender_recent_calls_to_target"] = senderProfile.RecentCallsToContract
	}

	// ====== 5. 触发 hook ======
	rules := s.ruleManager.GetRules()
	detections, err := s.hookManager.Trigger("contract_function_call", evalCtx, rules)
	if err != nil {
		return err
	}

	// 6. 持久化完整交易数据到 DB（用于交易浏览器）
	go s.saveTransactionToDB(ctx, &txData)

	// 7. 处理风险事件
	for _, detection := range detections {
		var matchedRule *ruleengine.Rule
		for _, rule := range rules {
			if rule.Metadata.Name == detection.RuleID {
				matchedRule = rule
				break
			}
		}

		if matchedRule == nil {
			continue
		}

		score, err := s.scorer.CalculateScore(matchedRule, evalCtx)
		if err != nil {
			s.logger.Error("Failed to calculate score", zap.Error(err))
			continue
		}

		detection.Score = score

		if err := s.executor.Execute(matchedRule, evalCtx, score); err != nil {
			s.logger.Error("Failed to execute actions", zap.Error(err))
		}

		s.logger.Info("Risk event detected",
			zap.String("rule", detection.RuleName),
			zap.String("tx_hash", detection.TxHash),
			zap.Int("score", score),
			zap.Int("call_depth", evalCtx.CallDepth),
			zap.Bool("privileged_call", privResult != nil && privResult.IsPrivilegedCall),
			zap.Bool("unauthorized", privResult != nil && privResult.IsPrivilegedCall && !privResult.CallerAuthorized))
	}

	return nil
}

// saveTransactionToDB 异步保存完整交易数据到 transactions 表（供交易浏览器查询）
func (s *RDSService) saveTransactionToDB(ctx context.Context, txData *models.TransactionData) {
	callStackJSON, _ := json.Marshal(txData.CallStack)
	eventsJSON, _ := json.Marshal(txData.Events)
	ts := time.Unix(int64(txData.Timestamp), 0)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO transactions (tx_hash, block_number, from_address, to_address, value, gas_price, gas_used,
		    input_data, status, timestamp, function_selector, call_stack, events_data, gas_limit, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		 ON CONFLICT DO NOTHING`,
		txData.TxHash, txData.BlockNumber, txData.FromAddress, txData.ToAddress,
		txData.Value, txData.GasPrice, txData.GasUsed,
		txData.InputData, txData.Status, ts,
		txData.FunctionSelector, callStackJSON, eventsJSON, txData.GasLimit)
	if err != nil {
		s.logger.Warn("Failed to save transaction to DB", zap.Error(err), zap.String("tx", txData.TxHash))
	}
}
