package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/models"
	"go.uber.org/zap"
)

// RiskEventRepository 风险事件仓储
type RiskEventRepository struct {
	db      *sql.DB
	redis   *cache.RedisClient
	logger  *zap.Logger
	writeCh chan *models.RiskEvent
	stopCh  chan struct{}
}

func NewRiskEventRepository(db *sql.DB, redis *cache.RedisClient, logger *zap.Logger) *RiskEventRepository {
	repo := &RiskEventRepository{
		db:      db,
		redis:   redis,
		logger:  logger,
		writeCh: make(chan *models.RiskEvent, 1000),
		stopCh:  make(chan struct{}),
	}

	// 启动异步写入 worker
	go repo.writeWorker()

	return repo
}

// writeWorker 异步写入处理
func (r *RiskEventRepository) writeWorker() {
	for {
		select {
		case event := <-r.writeCh:
			if err := r.writeToDBAndCache(context.Background(), event); err != nil {
				r.logger.Error("Failed to write risk event", zap.Error(err))
			}
		case <-r.stopCh:
			return
		}
	}
}

// writeToDBAndCache 实际写入逻辑
func (r *RiskEventRepository) writeToDBAndCache(ctx context.Context, event *models.RiskEvent) error {
	query := `INSERT INTO risk_events (event_type, severity, contract_address, tx_hash, description, score, detected_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		event.EventType, event.Severity, event.ContractAddress,
		event.TxHash, event.Description, event.Score, event.DetectedAt,
	).Scan(&event.ID)

	if err != nil {
		return err
	}

	// 缓存单个事件
	key := fmt.Sprintf("risk_event:%d", event.ID)
	r.redis.Set(ctx, key, event, 1*time.Hour)

	// 清除列表缓存
	r.redis.Delete(ctx, "risk_events:list")

	return nil
}

// Create 创建风险事件（异步）
func (r *RiskEventRepository) Create(ctx context.Context, event *models.RiskEvent) error {
	select {
	case r.writeCh <- event:
		return nil
	default:
		return fmt.Errorf("write queue full")
	}
}

// Close 关闭仓储
func (r *RiskEventRepository) Close() {
	close(r.stopCh)
}

// GetByID 获取单个事件（先缓存后 DB）
func (r *RiskEventRepository) GetByID(ctx context.Context, id int) (*models.RiskEvent, error) {
	key := fmt.Sprintf("risk_event:%d", id)

	// 尝试从缓存读取
	var event models.RiskEvent
	err := r.redis.Get(ctx, key, &event)
	if err == nil {
		return &event, nil
	}

	// 缓存未命中，从 DB 读取
	query := `SELECT id, event_type, severity, contract_address, tx_hash, description, score, detected_at
	          FROM risk_events WHERE id = $1`

	err = r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID, &event.EventType, &event.Severity, &event.ContractAddress,
		&event.TxHash, &event.Description, &event.Score, &event.DetectedAt,
	)

	if err != nil {
		return nil, err
	}

	// 写入缓存
	r.redis.Set(ctx, key, &event, 1*time.Hour)

	return &event, nil
}

// List 获取事件列表（先缓存后 DB）
func (r *RiskEventRepository) List(ctx context.Context, severity string, limit int) ([]*models.RiskEvent, error) {
	cacheKey := fmt.Sprintf("risk_events:list:%s:%d", severity, limit)

	// 尝试从缓存读取
	var events []*models.RiskEvent
	err := r.redis.Get(ctx, cacheKey, &events)
	if err == nil {
		return events, nil
	}

	// 从 DB 读取
	query := `SELECT id, event_type, severity, contract_address, tx_hash, description, score, detected_at
	          FROM risk_events WHERE 1=1`
	args := []interface{}{}

	if severity != "" {
		query += " AND severity = $1"
		args = append(args, severity)
	}

	query += fmt.Sprintf(" ORDER BY detected_at DESC LIMIT %d", limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event models.RiskEvent
		err := rows.Scan(
			&event.ID, &event.EventType, &event.Severity, &event.ContractAddress,
			&event.TxHash, &event.Description, &event.Score, &event.DetectedAt,
		)
		if err != nil {
			continue
		}
		events = append(events, &event)
	}

	// 写入缓存
	r.redis.Set(ctx, cacheKey, events, 5*time.Minute)

	return events, nil
}

// GetStats 获取统计数据（先缓存后 DB）
func (r *RiskEventRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	cacheKey := "risk_events:stats"

	// 尝试从缓存读取
	var stats map[string]interface{}
	data, err := r.redis.GetRaw(ctx, cacheKey)
	if err == nil {
		json.Unmarshal([]byte(data), &stats)
		return stats, nil
	}

	stats = make(map[string]interface{})

	// 总数
	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM risk_events").Scan(&total)
	stats["total"] = total

	// 按严重程度统计
	rows, _ := r.db.QueryContext(ctx, "SELECT severity, COUNT(*) FROM risk_events GROUP BY severity")
	defer rows.Close()

	severityCounts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		rows.Scan(&severity, &count)
		severityCounts[severity] = count
	}
	stats["by_severity"] = severityCounts

	// 写入缓存
	r.redis.Set(ctx, cacheKey, stats, 1*time.Minute)

	return stats, nil
}
