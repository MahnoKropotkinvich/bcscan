package profile

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// AddressProfile 地址行为画像（内存/Redis 表示）
type AddressProfile struct {
	Address string `json:"address"`

	// 滑动窗口统计（最近 N 分钟）
	RecentTxCount         int     `json:"recent_tx_count"`         // 最近窗口内的交易数
	RecentTargetContracts int     `json:"recent_target_contracts"` // 最近窗口内交互的不同合约数
	RecentTotalValue      float64 `json:"recent_total_value"`      // 最近窗口内的总转账金额(ETH)
	RecentFailedTxCount   int     `json:"recent_failed_tx_count"`  // 最近窗口内失败交易数
	RecentPrivilegeCalls  int     `json:"recent_privilege_calls"`  // 最近窗口内特权函数调用数

	// 针对特定合约的窗口统计
	RecentCallsToContract int `json:"recent_calls_to_contract"` // 最近窗口内对当前合约的调用数

	// 特权标记
	IsPrivileged   bool     `json:"is_privileged"`
	PrivilegeRoles []string `json:"privilege_roles"`

	// 累计
	TotalTxCount int64 `json:"total_tx_count"`
}

// Tracker 地址行为追踪器
// 使用 Redis Sorted Set 实现滑动窗口，score = timestamp
type Tracker struct {
	rdb    *redis.Client
	logger *zap.Logger
	window time.Duration // 滑动窗口大小，默认 10 分钟
}

// NewTracker 创建行为追踪器
func NewTracker(redisAddr string, logger *zap.Logger) *Tracker {
	rdb := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		DB:           1, // 使用 DB 1 隔离画像数据
		PoolSize:     5,
		MinIdleConns: 2,
	})
	return &Tracker{
		rdb:    rdb,
		logger: logger,
		window: 10 * time.Minute,
	}
}

// InteractionEvent 一次交互事件
type InteractionEvent struct {
	TxHash           string
	FromAddress      string
	ToAddress        string
	Value            string // wei
	GasUsed          uint64
	FunctionSelector string
	CallDepth        int
	HasDelegatecall  bool
	Status           uint64 // 0=failed, 1=success
	Timestamp        time.Time
}

// RecordInteraction 记录一次交互并更新滑动窗口
func (t *Tracker) RecordInteraction(ctx context.Context, event *InteractionEvent) error {
	now := float64(event.Timestamp.UnixMilli())
	pipe := t.rdb.Pipeline()

	from := strings.ToLower(event.FromAddress)
	to := strings.ToLower(event.ToAddress)

	// 编码事件数据: txHash|to|funcSel|value|status
	member := fmt.Sprintf("%s|%s|%s|%s|%d",
		event.TxHash, to, event.FunctionSelector, event.Value, event.Status)

	// 1. 发送方的交易历史 (sorted set, score=timestamp)
	txKey := fmt.Sprintf("profile:%s:txs", from)
	pipe.ZAdd(ctx, txKey, redis.Z{Score: now, Member: member})
	pipe.Expire(ctx, txKey, t.window+5*time.Minute) // 略大于窗口，保证窗口内数据完整

	// 2. 发送方对特定合约的调用历史
	if to != "" {
		pairKey := fmt.Sprintf("profile:%s:to:%s", from, to)
		pipe.ZAdd(ctx, pairKey, redis.Z{Score: now, Member: event.TxHash})
		pipe.Expire(ctx, pairKey, t.window+5*time.Minute)
	}

	// 3. 合约被调用历史（谁调用了这个合约）
	if to != "" {
		contractKey := fmt.Sprintf("profile:contract:%s:callers", to)
		callerMember := fmt.Sprintf("%s|%s|%s", from, event.TxHash, event.FunctionSelector)
		pipe.ZAdd(ctx, contractKey, redis.Z{Score: now, Member: callerMember})
		pipe.Expire(ctx, contractKey, t.window+5*time.Minute)
	}

	// 4. 特权函数调用计数
	if event.FunctionSelector != "" {
		privKey := fmt.Sprintf("profile:%s:priv_calls", from)
		pipe.ZAdd(ctx, privKey, redis.Z{Score: now, Member: event.TxHash + "|" + event.FunctionSelector})
		pipe.Expire(ctx, privKey, t.window+5*time.Minute)
	}

	// 5. 总交易计数 (简单 incr，不过期)
	countKey := fmt.Sprintf("profile:%s:total_tx", from)
	pipe.Incr(ctx, countKey)

	_, err := pipe.Exec(ctx)
	return err
}

// GetProfile 获取地址画像（滑动窗口统计）
func (t *Tracker) GetProfile(ctx context.Context, address string, targetContract string) (*AddressProfile, error) {
	addr := strings.ToLower(address)
	windowStart := float64(time.Now().Add(-t.window).UnixMilli())
	windowEnd := float64(time.Now().UnixMilli())

	profile := &AddressProfile{
		Address: addr,
	}

	// 1. 最近窗口内的所有交易
	txKey := fmt.Sprintf("profile:%s:txs", addr)
	txMembers, err := t.rdb.ZRangeByScore(ctx, txKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%.0f", windowStart),
		Max: fmt.Sprintf("%.0f", windowEnd),
	}).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	profile.RecentTxCount = len(txMembers)

	// 解析交易数据计算聚合指标
	contractSet := make(map[string]bool)
	for _, m := range txMembers {
		parts := strings.SplitN(m, "|", 5)
		if len(parts) >= 5 {
			to := parts[1]
			// funcSel := parts[2]
			valueStr := parts[3]
			status := parts[4]

			if to != "" {
				contractSet[to] = true
			}

			// 累加金额
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				profile.RecentTotalValue += val / 1e18 // wei -> ETH
			}

			// 失败交易
			if status == "0" {
				profile.RecentFailedTxCount++
			}
		}
	}
	profile.RecentTargetContracts = len(contractSet)

	// 2. 对特定合约的调用次数
	if targetContract != "" {
		pairKey := fmt.Sprintf("profile:%s:to:%s", addr, strings.ToLower(targetContract))
		count, err := t.rdb.ZCount(ctx, pairKey,
			fmt.Sprintf("%.0f", windowStart),
			fmt.Sprintf("%.0f", windowEnd)).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		profile.RecentCallsToContract = int(count)
	}

	// 3. 总交易数
	countKey := fmt.Sprintf("profile:%s:total_tx", addr)
	total, err := t.rdb.Get(ctx, countKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	profile.TotalTxCount = total

	// 4. 特权标记（从 Redis hash 读取）
	privInfoKey := fmt.Sprintf("profile:%s:priv_info", addr)
	privInfo, err := t.rdb.HGetAll(ctx, privInfoKey).Result()
	if err == nil && len(privInfo) > 0 {
		profile.IsPrivileged = privInfo["is_privileged"] == "true"
		if roles, ok := privInfo["roles"]; ok && roles != "" {
			profile.PrivilegeRoles = strings.Split(roles, ",")
		}
	}

	return profile, nil
}

// SetPrivileged 标记地址为特权地址
func (t *Tracker) SetPrivileged(ctx context.Context, address string, roles []string) error {
	addr := strings.ToLower(address)
	privInfoKey := fmt.Sprintf("profile:%s:priv_info", addr)
	return t.rdb.HSet(ctx, privInfoKey, map[string]interface{}{
		"is_privileged": "true",
		"roles":         strings.Join(roles, ","),
	}).Err()
}

// GetContractCallers 获取最近窗口内调用某合约的所有调用者
func (t *Tracker) GetContractCallers(ctx context.Context, contractAddress string) ([]string, error) {
	contract := strings.ToLower(contractAddress)
	windowStart := float64(time.Now().Add(-t.window).UnixMilli())
	windowEnd := float64(time.Now().UnixMilli())

	contractKey := fmt.Sprintf("profile:contract:%s:callers", contract)
	members, err := t.rdb.ZRangeByScore(ctx, contractKey, &redis.ZRangeBy{
		Min: fmt.Sprintf("%.0f", windowStart),
		Max: fmt.Sprintf("%.0f", windowEnd),
	}).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	callerSet := make(map[string]bool)
	for _, m := range members {
		parts := strings.SplitN(m, "|", 3)
		if len(parts) >= 1 {
			callerSet[parts[0]] = true
		}
	}

	callers := make([]string, 0, len(callerSet))
	for c := range callerSet {
		callers = append(callers, c)
	}
	return callers, nil
}

// PruneExpired 清理过期数据（可定期调用）
func (t *Tracker) PruneExpired(ctx context.Context, address string) error {
	addr := strings.ToLower(address)
	cutoff := fmt.Sprintf("%.0f", float64(time.Now().Add(-t.window-5*time.Minute).UnixMilli()))

	txKey := fmt.Sprintf("profile:%s:txs", addr)
	return t.rdb.ZRemRangeByScore(ctx, txKey, "-inf", cutoff).Err()
}

// Close 关闭连接
func (t *Tracker) Close() error {
	return t.rdb.Close()
}
