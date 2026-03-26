package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/haswell/bcscan/internal/auth"
	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/repository"
	"github.com/haswell/bcscan/internal/ruleengine"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// Config API 网关配置
type Config struct {
	DatabaseURL string
	Port        string
	RedisAddr   string
	RulesPath   string
	JWTSecret   string
}

func loadConfig() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://bcscan:bcscan_password@postgres:5432/bcscan?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RulesPath:   getEnv("RULES_PATH", "./rules/builtin"),
		JWTSecret:   getEnv("JWT_SECRET", "bcscan-dev-secret-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有来源
	},
}

// WebSocket 客户端管理
type WSHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

func newWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WSHub) run() {
	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true
		case conn := <-h.unregister:
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
		case message := <-h.broadcast:
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					delete(h.clients, conn)
					conn.Close()
				}
			}
		}
	}
}

func main() {
	cfg := loadConfig()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// 配置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	// Initialize components
	redis := cache.NewRedisClient(cfg.RedisAddr)
	riskRepo := repository.NewRiskEventRepository(db, redis, logger)
	ruleManager := ruleengine.NewRuleManager(cfg.RulesPath, redis, logger)
	authService := auth.NewService(db, cfg.JWTSecret, logger)

	// 初始加载规则
	if err := ruleManager.LoadRules(context.Background()); err != nil {
		logger.Warn("Failed to load rules on startup", zap.Error(err))
	}

	// WebSocket Hub
	hub := newWSHub()
	go hub.run()

	// 启动 Redis 订阅，监听新的风险事件用于 WebSocket 推送
	go subscribeRiskEvents(redis, hub, logger)

	logger.Info("API Gateway starting", zap.String("port", cfg.Port))

	r := mux.NewRouter()
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware(logger))

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// 认证 API（无需登录）
	api.HandleFunc("/auth/register", handleRegister(authService)).Methods("POST")
	api.HandleFunc("/auth/login", handleLogin(authService)).Methods("POST")

	// 风险事件 API
	api.HandleFunc("/risks", getRiskEvents(riskRepo)).Methods("GET")
	api.HandleFunc("/risks/{id}", getRiskEvent(riskRepo)).Methods("GET")

	// 统计 API
	api.HandleFunc("/stats", getStats(riskRepo)).Methods("GET")
	api.HandleFunc("/stats/trend", getTrend(riskRepo, db)).Methods("GET")

	// 规则管理 API
	api.HandleFunc("/rules", getRules(ruleManager)).Methods("GET")

	// 需要认证的 API
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authService.AuthMiddleware)
	protected.HandleFunc("/rules/reload", reloadRules(ruleManager)).Methods("POST")
	protected.HandleFunc("/auth/me", handleGetMe(authService)).Methods("GET")

	// 系统状态 API
	api.HandleFunc("/health", healthCheck(db, redis)).Methods("GET")

	// WebSocket
	api.HandleFunc("/ws", handleWebSocket(hub))

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}

// subscribeRiskEvents 订阅 Redis 频道，将新风险事件推送到 WebSocket
func subscribeRiskEvents(redis *cache.RedisClient, hub *WSHub, logger *zap.Logger) {
	ctx := context.Background()
	pubsub := redis.Subscribe(ctx, "risk_events:new")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		hub.broadcast <- []byte(msg.Payload)
	}
}

// ==================== Middleware ====================

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(logger *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Debug("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)))
		})
	}
}

// ==================== Response Helpers ====================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Total   int         `json:"total,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, APIResponse{Success: false, Error: message})
}

// ==================== Risk Events API ====================

func getRiskEvents(repo *repository.RiskEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		severity := r.URL.Query().Get("severity")
		search := r.URL.Query().Get("search")

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}

		pageSize := 20
		if ps := r.URL.Query().Get("page_size"); ps != "" {
			if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
				pageSize = parsed
			}
		}

		// 兼容旧的 limit 参数
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				pageSize = parsed
			}
		}

		result, err := repo.ListPaged(r.Context(), severity, search, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

func getRiskEvent(repo *repository.RiskEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid ID")
			return
		}

		event, err := repo.GetByID(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusNotFound, "Risk event not found")
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: event})
	}
}

// ==================== Statistics API ====================

func getStats(repo *repository.RiskEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := repo.GetStats(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
	}
}

// TrendPoint 趋势数据点
type TrendPoint struct {
	Time     string `json:"time"`
	Critical int    `json:"critical"`
	High     int    `json:"high"`
	Medium   int    `json:"medium"`
	Low      int    `json:"low"`
}

func getTrend(repo *repository.RiskEventRepository, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeRange := r.URL.Query().Get("range")
		if timeRange == "" {
			timeRange = "24h"
		}

		var interval string
		var duration string
		var format string

		switch timeRange {
		case "10m":
			interval = "1 minute"
			duration = "10 minutes"
			format = "HH24:MI"
		case "1h":
			interval = "5 minutes"
			duration = "1 hour"
			format = "HH24:MI"
		case "6h":
			interval = "30 minutes"
			duration = "6 hours"
			format = "HH24:MI"
		case "24h":
			interval = "1 hour"
			duration = "24 hours"
			format = "MM-DD HH24:00"
		case "7d":
			interval = "1 day"
			duration = "7 days"
			format = "MM-DD"
		default:
			interval = "1 hour"
			duration = "24 hours"
			format = "MM-DD HH24:00"
		}

		query := `
			WITH time_series AS (
				SELECT generate_series(
					date_trunc('hour', NOW() - $1::interval),
					NOW(),
					$2::interval
				) AS bucket
			)
			SELECT
				to_char(ts.bucket, $3) AS time,
				COALESCE(SUM(CASE WHEN r.severity = 'critical' THEN 1 ELSE 0 END), 0) AS critical,
				COALESCE(SUM(CASE WHEN r.severity = 'high' THEN 1 ELSE 0 END), 0) AS high,
				COALESCE(SUM(CASE WHEN r.severity = 'medium' THEN 1 ELSE 0 END), 0) AS medium,
				COALESCE(SUM(CASE WHEN r.severity = 'low' THEN 1 ELSE 0 END), 0) AS low
			FROM time_series ts
			LEFT JOIN risk_events r ON r.detected_at >= ts.bucket
				AND r.detected_at < ts.bucket + $2::interval
			GROUP BY ts.bucket
			ORDER BY ts.bucket`

		rows, err := db.QueryContext(r.Context(), query, duration, interval, format)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var trend []TrendPoint
		for rows.Next() {
			var p TrendPoint
			if err := rows.Scan(&p.Time, &p.Critical, &p.High, &p.Medium, &p.Low); err != nil {
				continue
			}
			trend = append(trend, p)
		}

		if trend == nil {
			trend = []TrendPoint{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: trend})
	}
}

// ==================== Rules API ====================

func getRules(rm *ruleengine.RuleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules := rm.GetRules()
		respondJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    rules,
			Total:   len(rules),
		})
	}
}

func reloadRules(rm *ruleengine.RuleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		// 清除 Redis 缓存，强制从文件重新加载
		rm.InvalidateCache(ctx)

		if err := rm.LoadRules(ctx); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := rm.PublishUpdate(ctx); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"message": "Rules reloaded successfully",
				"count":   len(rm.GetRules()),
			},
		})
	}
}

// ==================== System API ====================

func healthCheck(db *sql.DB, redis *cache.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"services":  map[string]string{},
		}

		services := health["services"].(map[string]string)

		// 检查数据库
		if err := db.Ping(); err != nil {
			services["database"] = "error: " + err.Error()
			health["status"] = "degraded"
		} else {
			services["database"] = "ok"
		}

		// 检查 Redis
		ctx := context.Background()
		if err := redis.Set(ctx, "health:check", "ok", 10*time.Second); err != nil {
			services["redis"] = "error: " + err.Error()
			health["status"] = "degraded"
		} else {
			services["redis"] = "ok"
		}

		status := http.StatusOK
		if health["status"] == "degraded" {
			status = http.StatusServiceUnavailable
		}

		respondJSON(w, status, health)
	}
}

// ==================== WebSocket ====================

func handleWebSocket(hub *WSHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		hub.register <- conn

		// 保持连接，等待客户端断开
		go func() {
			defer func() {
				hub.unregister <- conn
			}()
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()
	}
}

// ==================== Auth API ====================

func handleRegister(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req auth.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误")
			return
		}

		user, err := svc.Register(r.Context(), &req)
		if err != nil {
			status := http.StatusInternalServerError
			if err == auth.ErrUserExists {
				status = http.StatusConflict
			}
			respondError(w, status, err.Error())
			return
		}

		respondJSON(w, http.StatusCreated, APIResponse{Success: true, Data: user})
	}
}

func handleLogin(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req auth.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误")
			return
		}

		resp, err := svc.Login(r.Context(), &req)
		if err != nil {
			status := http.StatusInternalServerError
			if err == auth.ErrInvalidCredentials {
				status = http.StatusUnauthorized
			}
			respondError(w, status, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: resp})
	}
}

func handleGetMe(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		if user == nil {
			respondError(w, http.StatusUnauthorized, "未登录")
			return
		}

		// 查询完整用户信息
		fullUser, err := svc.GetCurrentUser(r.Context(), user.ID)
		if err != nil {
			respondError(w, http.StatusNotFound, "用户不存在")
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: fullUser})
	}
}
