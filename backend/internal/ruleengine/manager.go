package ruleengine

import (
	"context"
	"time"

	"github.com/haswell/bcscan/internal/cache"
	"go.uber.org/zap"
)

const (
	RulesCacheKey      = "rules:all"
	RulesUpdateChannel = "rules:update"
)

// RuleManager 规则管理器（支持热加载）
type RuleManager struct {
	loader *RuleLoader
	redis  *cache.RedisClient
	logger *zap.Logger
	rules  []*Rule
}

func NewRuleManager(rulesDir string, redis *cache.RedisClient, logger *zap.Logger) *RuleManager {
	return &RuleManager{
		loader: NewRuleLoader(rulesDir, logger),
		redis:  redis,
		logger: logger,
		rules:  []*Rule{},
	}
}

// LoadRules 加载规则（优先从 Redis）
func (rm *RuleManager) LoadRules(ctx context.Context) error {
	// 尝试从 Redis 加载
	var cachedRules []*Rule
	err := rm.redis.Get(ctx, RulesCacheKey, &cachedRules)
	if err == nil && len(cachedRules) > 0 {
		rm.rules = cachedRules
		rm.logger.Info("Loaded rules from Redis", zap.Int("count", len(cachedRules)))
		return nil
	}

	// Redis 没有，从文件加载
	if err := rm.loader.LoadAll(); err != nil {
		return err
	}

	rm.rules = rm.loader.GetEnabledRules()

	// 缓存到 Redis
	if err := rm.redis.Set(ctx, RulesCacheKey, rm.rules, 0); err != nil {
		rm.logger.Warn("Failed to cache rules to Redis", zap.Error(err))
	}

	rm.logger.Info("Loaded rules from files", zap.Int("count", len(rm.rules)))
	return nil
}

// GetRules 获取当前规则
func (rm *RuleManager) GetRules() []*Rule {
	return rm.rules
}

// SubscribeUpdates 订阅规则更新
func (rm *RuleManager) SubscribeUpdates(ctx context.Context) {
	pubsub := rm.redis.Subscribe(ctx, RulesUpdateChannel)
	defer pubsub.Close()

	rm.logger.Info("Subscribed to rule updates")

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			rm.logger.Info("Received rule update notification", zap.String("message", msg.Payload))
			if err := rm.LoadRules(ctx); err != nil {
				rm.logger.Error("Failed to reload rules", zap.Error(err))
			} else {
				rm.logger.Info("Rules reloaded successfully", zap.Int("count", len(rm.rules)))
			}
		}
	}
}

// PublishUpdate 发布规则更新通知
func (rm *RuleManager) PublishUpdate(ctx context.Context) error {
	return rm.redis.Publish(ctx, RulesUpdateChannel, map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"action":    "reload",
	})
}
