package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/haswell/bcscan/internal/auth"
	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/config"
	"github.com/haswell/bcscan/internal/models"
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
		DatabaseURL: config.GetEnv("DATABASE_URL", "postgres://bcscan:bcscan_password@postgres:5432/bcscan?sslmode=disable"),
		Port:        config.GetEnv("PORT", "8080"),
		RedisAddr:   config.GetEnv("REDIS_ADDR", "localhost:6379"),
		RulesPath:   config.GetEnv("RULES_PATH", "./rules/builtin"),
		JWTSecret:   config.GetEnv("JWT_SECRET", "bcscan-dev-secret-change-in-production"),
	}
}

// getClientIP 获取客户端 IP
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// 去掉端口
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHub WebSocket 客户端管理
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

	// ==================== 公开 API（无需登录） ====================
	api.HandleFunc("/auth/register", handleRegister(authService)).Methods("POST")
	api.HandleFunc("/auth/login", handleLogin(authService, db)).Methods("POST")

	// 风险事件 API（查看权限公开）
	api.HandleFunc("/risks", getRiskEvents(riskRepo)).Methods("GET")
	api.HandleFunc("/risks/{id}", getRiskEvent(riskRepo)).Methods("GET")

	// 统计 API
	api.HandleFunc("/stats", getStats(riskRepo)).Methods("GET")
	api.HandleFunc("/stats/trend", getTrend(riskRepo, db)).Methods("GET")

	// 规则管理 API（查看公开）
	api.HandleFunc("/rules", getRules(ruleManager)).Methods("GET")

	// 系统状态 API
	api.HandleFunc("/health", healthCheck(db, redis)).Methods("GET")

	// WebSocket
	api.HandleFunc("/ws", handleWebSocket(hub))

	// ==================== 需要认证的 API ====================
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authService.AuthMiddleware)

	// 个人信息
	protected.HandleFunc("/auth/me", handleGetMe(authService)).Methods("GET")

	// 规则热加载（需要 analyst/admin）
	analystRoutes := protected.PathPrefix("").Subrouter()
	analystRoutes.Use(authService.RequireRole(auth.RoleAnalyst, auth.RoleDeveloper))
	analystRoutes.HandleFunc("/rules/reload", reloadRules(ruleManager, authService)).Methods("POST")

	// ==================== 告警管理 API ====================
	protected.HandleFunc("/alerts", getAlerts(db)).Methods("GET")
	protected.HandleFunc("/alerts/stats", getAlertStats(db)).Methods("GET")
	protected.HandleFunc("/alerts/{id}", getAlertDetail(db)).Methods("GET")

	alertActionRoutes := protected.PathPrefix("").Subrouter()
	alertActionRoutes.Use(authService.RequireRole(auth.RoleAnalyst, auth.RoleDeveloper, auth.RoleOperator))
	alertActionRoutes.HandleFunc("/alerts/{id}/acknowledge", acknowledgeAlert(db, authService)).Methods("POST")
	alertActionRoutes.HandleFunc("/alerts/{id}/resolve", resolveAlert(db, authService)).Methods("POST")
	alertActionRoutes.HandleFunc("/alerts/{id}/ignore", ignoreAlert(db, authService)).Methods("POST")
	alertActionRoutes.HandleFunc("/alerts/{id}/note", addAlertNote(db, authService)).Methods("POST")

	// ==================== 报告 API ====================
	protected.HandleFunc("/reports/summary", getReportSummary(db)).Methods("GET")
	protected.HandleFunc("/reports/by-type", getReportByType(db)).Methods("GET")
	protected.HandleFunc("/reports/by-contract", getReportByContract(db)).Methods("GET")
	protected.HandleFunc("/reports/timeline", getReportTimeline(db)).Methods("GET")
	protected.HandleFunc("/reports/export", exportReport(db)).Methods("GET")

	// ==================== 用户管理 API（仅 admin） ====================
	adminRoutes := protected.PathPrefix("").Subrouter()
	adminRoutes.Use(authService.RequireRole()) // 空参数 = 仅 admin

	adminRoutes.HandleFunc("/users", listUsers(authService)).Methods("GET")
	adminRoutes.HandleFunc("/users/roles", getRolesInfo(authService)).Methods("GET")
	adminRoutes.HandleFunc("/users/{id}/role", updateUserRole(authService)).Methods("PUT")
	adminRoutes.HandleFunc("/users/{id}/status", updateUserStatus(authService)).Methods("PUT")

	// ==================== 审计日志 API（admin + operator） ====================
	auditRoutes := protected.PathPrefix("").Subrouter()
	auditRoutes.Use(authService.RequireRole(auth.RoleOperator))

	auditRoutes.HandleFunc("/audit-logs", listAuditLogs(authService)).Methods("GET")
	auditRoutes.HandleFunc("/audit-logs/actions", getAuditActions(authService)).Methods("GET")

	// ==================== 交易浏览器 API ====================
	api.HandleFunc("/explorer/tx/{hash}", getTransactionByHash(db)).Methods("GET")
	api.HandleFunc("/explorer/address/{address}/txs", getTransactionsByAddress(db)).Methods("GET")
	api.HandleFunc("/explorer/address/{address}/summary", getAddressSummary(db)).Methods("GET")
	api.HandleFunc("/explorer/tx/{hash}/risks", getRisksByTxHash(db)).Methods("GET")
	api.HandleFunc("/explorer/tx/{hash}/alerts", getAlertsByTxHash(db)).Methods("GET")
	api.HandleFunc("/explorer/decode/{selector}", decodeFunctionSelector(db)).Methods("GET")
	api.HandleFunc("/explorer/recent", getRecentTransactions(db)).Methods("GET")

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

		var interval, duration, format string

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

func reloadRules(rm *ruleengine.RuleManager, authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		rm.InvalidateCache(ctx)

		if err := rm.LoadRules(ctx); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := rm.PublishUpdate(ctx); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 记录审计日志
		user := auth.GetUserFromContext(r.Context())
		if user != nil {
			authSvc.RecordAuditLog(r.Context(), &user.ID, "rule_reload", "/api/rules/reload",
				map[string]int{"count": len(rm.GetRules())}, getClientIP(r))
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

		if err := db.Ping(); err != nil {
			services["database"] = "error: " + err.Error()
			health["status"] = "degraded"
		} else {
			services["database"] = "ok"
		}

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

		// 记录审计日志
		svc.RecordAuditLog(r.Context(), &user.ID, "user_register", "/api/auth/register",
			map[string]string{"username": user.Username, "role": user.Role}, getClientIP(r))

		respondJSON(w, http.StatusCreated, APIResponse{Success: true, Data: user})
	}
}

func handleLogin(svc *auth.Service, db *sql.DB) http.HandlerFunc {
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
			// 记录登录失败
			svc.RecordAuditLog(r.Context(), nil, "login_failed", "/api/auth/login",
				map[string]string{"username": req.Username}, getClientIP(r))
			respondError(w, status, err.Error())
			return
		}

		// 记录登录成功
		svc.RecordAuditLog(r.Context(), &resp.User.ID, "login_success", "/api/auth/login",
			map[string]string{"username": resp.User.Username}, getClientIP(r))

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

		fullUser, err := svc.GetCurrentUser(r.Context(), user.ID)
		if err != nil {
			respondError(w, http.StatusNotFound, "用户不存在")
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: fullUser})
	}
}

// ==================== 用户管理 API ====================

func listUsers(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.URL.Query().Get("role")
		search := r.URL.Query().Get("search")
		page := parseIntDefault(r.URL.Query().Get("page"), 1)
		pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 20)

		result, err := svc.ListUsers(r.Context(), role, search, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
	}
}

func getRolesInfo(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roles := svc.GetRolesInfo(r.Context())
		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: roles})
	}
}

func updateUserRole(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的用户 ID")
			return
		}

		var req struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误")
			return
		}

		if err := svc.UpdateUserRole(r.Context(), id, req.Role); err != nil {
			status := http.StatusInternalServerError
			if err == auth.ErrUserNotFound {
				status = http.StatusNotFound
			} else if err == auth.ErrInvalidRole {
				status = http.StatusBadRequest
			}
			respondError(w, status, err.Error())
			return
		}

		// 记录审计日志
		operator := auth.GetUserFromContext(r.Context())
		if operator != nil {
			svc.RecordAuditLog(r.Context(), &operator.ID, "user_role_change", fmt.Sprintf("/api/users/%d/role", id),
				map[string]interface{}{"target_user_id": id, "new_role": req.Role}, getClientIP(r))
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "角色更新成功"}})
	}
}

func updateUserStatus(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的用户 ID")
			return
		}

		var req struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "请求格式错误")
			return
		}

		if err := svc.UpdateUserStatus(r.Context(), id, req.Status); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// 记录审计日志
		operator := auth.GetUserFromContext(r.Context())
		if operator != nil {
			svc.RecordAuditLog(r.Context(), &operator.ID, "user_status_change", fmt.Sprintf("/api/users/%d/status", id),
				map[string]interface{}{"target_user_id": id, "new_status": req.Status}, getClientIP(r))
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "用户状态更新成功"}})
	}
}

// ==================== 告警管理 API ====================

func getAlerts(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		severity := r.URL.Query().Get("severity")
		search := r.URL.Query().Get("search")
		page := parseIntDefault(r.URL.Query().Get("page"), 1)
		pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 20)

		where := "WHERE 1=1"
		args := []interface{}{}
		argIdx := 1

		if status != "" && status != "all" {
			where += fmt.Sprintf(" AND a.status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}
		if severity != "" && severity != "all" {
			where += fmt.Sprintf(" AND a.severity = $%d", argIdx)
			args = append(args, severity)
			argIdx++
		}
		if search != "" {
			where += fmt.Sprintf(" AND (a.title ILIKE $%d OR a.message ILIKE $%d OR r.tx_hash ILIKE $%d)", argIdx, argIdx, argIdx)
			args = append(args, "%"+search+"%")
			argIdx++
		}

		// 总数
		var total int
		countQ := fmt.Sprintf("SELECT COUNT(*) FROM alerts a LEFT JOIN risk_events r ON a.risk_event_id = r.id %s", where)
		db.QueryRowContext(r.Context(), countQ, args...).Scan(&total)

		offset := (page - 1) * pageSize
		pages := total / pageSize
		if total%pageSize > 0 {
			pages++
		}

		dataQ := fmt.Sprintf(
			`SELECT a.id, a.risk_event_id, a.title, COALESCE(a.message,''), a.severity, a.status,
			        a.assigned_to, a.acknowledged_at, a.acknowledged_by, a.resolved_at, a.resolved_by,
			        a.notes, a.created_at,
			        COALESCE(r.tx_hash,''), COALESCE(r.contract_address,''), COALESCE(r.score,0)
			 FROM alerts a
			 LEFT JOIN risk_events r ON a.risk_event_id = r.id
			 %s ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d`,
			where, argIdx, argIdx+1)
		args = append(args, pageSize, offset)

		rows, err := db.QueryContext(r.Context(), dataQ, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var alerts []models.Alert
		for rows.Next() {
			var a models.Alert
			if err := rows.Scan(&a.ID, &a.RiskEventID, &a.Title, &a.Message, &a.Severity, &a.Status,
				&a.AssignedTo, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.ResolvedBy,
				&a.Notes, &a.CreatedAt,
				&a.TxHash, &a.ContractAddress, &a.Score); err != nil {
				continue
			}
			alerts = append(alerts, a)
		}
		if alerts == nil {
			alerts = []models.Alert{}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items":     alerts,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
		})
	}
}

func getAlertStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]interface{}{}

		// 按状态统计
		rows, _ := db.QueryContext(r.Context(),
			"SELECT status, COUNT(*) FROM alerts GROUP BY status")
		if rows != nil {
			defer rows.Close()
			byStatus := map[string]int{}
			for rows.Next() {
				var s string
				var c int
				rows.Scan(&s, &c)
				byStatus[s] = c
			}
			stats["by_status"] = byStatus
		}

		// 未处理告警数
		var pending int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM alerts WHERE status = 'pending'").Scan(&pending)
		stats["pending_count"] = pending

		// 总数
		var total int
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM alerts").Scan(&total)
		stats["total"] = total

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: stats})
	}
}

func getAlertDetail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "无效的告警 ID")
			return
		}

		// 查询告警
		var a models.Alert
		err = db.QueryRowContext(r.Context(),
			`SELECT a.id, a.risk_event_id, a.title, COALESCE(a.message,''), a.severity, a.status,
			        a.assigned_to, a.acknowledged_at, a.acknowledged_by, a.resolved_at, a.resolved_by,
			        a.notes, a.created_at,
			        COALESCE(r.tx_hash,''), COALESCE(r.contract_address,''), COALESCE(r.score,0)
			 FROM alerts a
			 LEFT JOIN risk_events r ON a.risk_event_id = r.id
			 WHERE a.id = $1`, id,
		).Scan(&a.ID, &a.RiskEventID, &a.Title, &a.Message, &a.Severity, &a.Status,
			&a.AssignedTo, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.ResolvedBy,
			&a.Notes, &a.CreatedAt,
			&a.TxHash, &a.ContractAddress, &a.Score)
		if err != nil {
			respondError(w, http.StatusNotFound, "告警不存在")
			return
		}

		// 查询处理历史
		histRows, _ := db.QueryContext(r.Context(),
			`SELECT h.id, h.alert_id, h.user_id, COALESCE(u.username,'系统'), h.action,
			        COALESCE(h.old_status,''), COALESCE(h.new_status,''), COALESCE(h.note,''), h.created_at
			 FROM alert_history h
			 LEFT JOIN users u ON h.user_id = u.id
			 WHERE h.alert_id = $1 ORDER BY h.created_at`, id)

		var history []models.AlertHistory
		if histRows != nil {
			defer histRows.Close()
			for histRows.Next() {
				var h models.AlertHistory
				if err := histRows.Scan(&h.ID, &h.AlertID, &h.UserID, &h.Username, &h.Action,
					&h.OldStatus, &h.NewStatus, &h.Note, &h.CreatedAt); err != nil {
					continue
				}
				history = append(history, h)
			}
		}
		if history == nil {
			history = []models.AlertHistory{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
			"alert":   a,
			"history": history,
		}})
	}
}

func acknowledgeAlert(db *sql.DB, authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		user := auth.GetUserFromContext(r.Context())

		// 更新告警
		result, err := db.ExecContext(r.Context(),
			`UPDATE alerts SET status = 'acknowledged', acknowledged_at = NOW(), 
			 acknowledged_by = $1, updated_at = NOW() WHERE id = $2 AND status = 'pending'`,
			user.ID, id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			respondError(w, http.StatusBadRequest, "告警不存在或状态不允许确认")
			return
		}

		// 记录历史
		db.ExecContext(r.Context(),
			`INSERT INTO alert_history (alert_id, user_id, action, old_status, new_status, created_at)
			 VALUES ($1, $2, 'acknowledged', 'pending', 'acknowledged', NOW())`,
			id, user.ID)

		// 审计日志
		authSvc.RecordAuditLog(r.Context(), &user.ID, "alert_acknowledge", fmt.Sprintf("/api/alerts/%d", id), nil, getClientIP(r))

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "告警已确认"}})
	}
}

func resolveAlert(db *sql.DB, authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		user := auth.GetUserFromContext(r.Context())

		var req struct {
			Note string `json:"note"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// 获取旧状态
		var oldStatus string
		db.QueryRowContext(r.Context(), "SELECT status FROM alerts WHERE id = $1", id).Scan(&oldStatus)

		result, err := db.ExecContext(r.Context(),
			`UPDATE alerts SET status = 'resolved', resolved_at = NOW(), resolved_by = $1,
			 notes = COALESCE(notes, '') || CASE WHEN $3 != '' THEN E'\n' || $3 ELSE '' END,
			 updated_at = NOW() WHERE id = $2 AND status IN ('pending', 'acknowledged')`,
			user.ID, id, req.Note)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			respondError(w, http.StatusBadRequest, "告警不存在或状态不允许解决")
			return
		}

		db.ExecContext(r.Context(),
			`INSERT INTO alert_history (alert_id, user_id, action, old_status, new_status, note, created_at)
			 VALUES ($1, $2, 'resolved', $3, 'resolved', $4, NOW())`,
			id, user.ID, oldStatus, req.Note)

		authSvc.RecordAuditLog(r.Context(), &user.ID, "alert_resolve", fmt.Sprintf("/api/alerts/%d", id),
			map[string]string{"note": req.Note}, getClientIP(r))

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "告警已解决"}})
	}
}

func ignoreAlert(db *sql.DB, authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		user := auth.GetUserFromContext(r.Context())

		var req struct {
			Note string `json:"note"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var oldStatus string
		db.QueryRowContext(r.Context(), "SELECT status FROM alerts WHERE id = $1", id).Scan(&oldStatus)

		result, err := db.ExecContext(r.Context(),
			`UPDATE alerts SET status = 'ignored', resolved_at = NOW(), resolved_by = $1,
			 notes = COALESCE(notes, '') || CASE WHEN $3 != '' THEN E'\n' || $3 ELSE '' END,
			 updated_at = NOW() WHERE id = $2 AND status IN ('pending', 'acknowledged')`,
			user.ID, id, req.Note)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			respondError(w, http.StatusBadRequest, "告警不存在或状态不允许忽略")
			return
		}

		db.ExecContext(r.Context(),
			`INSERT INTO alert_history (alert_id, user_id, action, old_status, new_status, note, created_at)
			 VALUES ($1, $2, 'ignored', $3, 'ignored', $4, NOW())`,
			id, user.ID, oldStatus, req.Note)

		authSvc.RecordAuditLog(r.Context(), &user.ID, "alert_ignore", fmt.Sprintf("/api/alerts/%d", id),
			map[string]string{"note": req.Note}, getClientIP(r))

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "告警已忽略"}})
	}
}

func addAlertNote(db *sql.DB, authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		user := auth.GetUserFromContext(r.Context())

		var req struct {
			Note string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Note == "" {
			respondError(w, http.StatusBadRequest, "备注不能为空")
			return
		}

		_, err := db.ExecContext(r.Context(),
			`UPDATE alerts SET notes = COALESCE(notes, '') || E'\n[`+user.Username+`] ' || $1,
			 updated_at = NOW() WHERE id = $2`,
			req.Note, id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		db.ExecContext(r.Context(),
			`INSERT INTO alert_history (alert_id, user_id, action, note, created_at)
			 VALUES ($1, $2, 'note_added', $3, NOW())`,
			id, user.ID, req.Note)

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "备注已添加"}})
	}
}

// ==================== 报告 API ====================

func getReportSummary(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeRange := r.URL.Query().Get("range")
		if timeRange == "" {
			timeRange = "7d"
		}

		var duration string
		switch timeRange {
		case "24h":
			duration = "24 hours"
		case "7d":
			duration = "7 days"
		case "30d":
			duration = "30 days"
		case "90d":
			duration = "90 days"
		default:
			duration = "7 days"
		}

		summary := map[string]interface{}{}

		// 总事件数
		var total int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM risk_events WHERE detected_at > NOW() - $1::interval", duration).Scan(&total)
		summary["total_events"] = total

		// 按严重程度
		rows1, _ := db.QueryContext(r.Context(),
			"SELECT severity, COUNT(*) FROM risk_events WHERE detected_at > NOW() - $1::interval GROUP BY severity ORDER BY COUNT(*) DESC", duration)
		if rows1 != nil {
			defer rows1.Close()
			var bySeverity []map[string]interface{}
			for rows1.Next() {
				var s string
				var c int
				rows1.Scan(&s, &c)
				bySeverity = append(bySeverity, map[string]interface{}{"severity": s, "count": c})
			}
			summary["by_severity"] = bySeverity
		}

		// 平均分数
		var avgScore float64
		db.QueryRowContext(r.Context(),
			"SELECT COALESCE(AVG(score), 0) FROM risk_events WHERE detected_at > NOW() - $1::interval", duration).Scan(&avgScore)
		summary["avg_score"] = fmt.Sprintf("%.1f", avgScore)

		// 告警处理率
		var totalAlerts, resolvedAlerts int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM alerts WHERE created_at > NOW() - $1::interval", duration).Scan(&totalAlerts)
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM alerts WHERE created_at > NOW() - $1::interval AND status IN ('resolved', 'ignored')", duration).Scan(&resolvedAlerts)
		if totalAlerts > 0 {
			summary["alert_resolve_rate"] = fmt.Sprintf("%.1f%%", float64(resolvedAlerts)/float64(totalAlerts)*100)
		} else {
			summary["alert_resolve_rate"] = "N/A"
		}
		summary["total_alerts"] = totalAlerts
		summary["resolved_alerts"] = resolvedAlerts

		summary["time_range"] = timeRange
		summary["generated_at"] = time.Now().Format(time.RFC3339)

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: summary})
	}
}

func getReportByType(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeRange := r.URL.Query().Get("range")
		duration := rangeToDuration(timeRange)

		rows, err := db.QueryContext(r.Context(),
			`SELECT event_type, severity, COUNT(*), COALESCE(AVG(score), 0)
			 FROM risk_events WHERE detected_at > NOW() - $1::interval
			 GROUP BY event_type, severity ORDER BY COUNT(*) DESC`, duration)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var eventType, severity string
			var count int
			var avgScore float64
			rows.Scan(&eventType, &severity, &count, &avgScore)
			result = append(result, map[string]interface{}{
				"event_type": eventType,
				"severity":   severity,
				"count":      count,
				"avg_score":  fmt.Sprintf("%.1f", avgScore),
			})
		}
		if result == nil {
			result = []map[string]interface{}{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
	}
}

func getReportByContract(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeRange := r.URL.Query().Get("range")
		duration := rangeToDuration(timeRange)
		limitStr := r.URL.Query().Get("limit")
		limit := parseIntDefault(limitStr, 10)

		rows, err := db.QueryContext(r.Context(),
			`SELECT contract_address, COUNT(*) as cnt, 
			        COALESCE(AVG(score), 0),
			        MAX(severity)
			 FROM risk_events 
			 WHERE detected_at > NOW() - $1::interval AND contract_address != ''
			 GROUP BY contract_address ORDER BY cnt DESC LIMIT $2`, duration, limit)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var addr, maxSeverity string
			var count int
			var avgScore float64
			rows.Scan(&addr, &count, &avgScore, &maxSeverity)
			result = append(result, map[string]interface{}{
				"contract_address": addr,
				"event_count":      count,
				"avg_score":        fmt.Sprintf("%.1f", avgScore),
				"max_severity":     maxSeverity,
			})
		}
		if result == nil {
			result = []map[string]interface{}{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
	}
}

func getReportTimeline(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeRange := r.URL.Query().Get("range")
		duration := rangeToDuration(timeRange)

		var interval, format string
		switch timeRange {
		case "24h":
			interval = "1 hour"
			format = "MM-DD HH24:00"
		case "30d":
			interval = "1 day"
			format = "MM-DD"
		case "90d":
			interval = "3 days"
			format = "MM-DD"
		default: // 7d
			interval = "1 day"
			format = "MM-DD"
		}

		rows, err := db.QueryContext(r.Context(),
			`WITH time_series AS (
				SELECT generate_series(
					date_trunc('day', NOW() - $1::interval),
					NOW(),
					$2::interval
				) AS bucket
			)
			SELECT
				to_char(ts.bucket, $3) AS time,
				COALESCE(SUM(CASE WHEN r.severity = 'critical' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN r.severity = 'high' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN r.severity = 'medium' THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN r.severity = 'low' THEN 1 ELSE 0 END), 0)
			FROM time_series ts
			LEFT JOIN risk_events r ON r.detected_at >= ts.bucket AND r.detected_at < ts.bucket + $2::interval
			GROUP BY ts.bucket
			ORDER BY ts.bucket`, duration, interval, format)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			var t string
			var critical, high, medium, low int
			rows.Scan(&t, &critical, &high, &medium, &low)
			result = append(result, map[string]interface{}{
				"time":     t,
				"critical": critical,
				"high":     high,
				"medium":   medium,
				"low":      low,
				"total":    critical + high + medium + low,
			})
		}
		if result == nil {
			result = []map[string]interface{}{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
	}
}

func exportReport(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeRange := r.URL.Query().Get("range")
		duration := rangeToDuration(timeRange)

		report := map[string]interface{}{
			"report_type":  "risk_analysis",
			"generated_at": time.Now().Format(time.RFC3339),
			"time_range":   timeRange,
		}

		// 汇总
		var total int
		var avgScore float64
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*), COALESCE(AVG(score),0) FROM risk_events WHERE detected_at > NOW() - $1::interval", duration).Scan(&total, &avgScore)
		report["total_events"] = total
		report["avg_score"] = avgScore

		// 按类型统计
		rows, _ := db.QueryContext(r.Context(),
			"SELECT event_type, COUNT(*), COALESCE(AVG(score),0) FROM risk_events WHERE detected_at > NOW() - $1::interval GROUP BY event_type ORDER BY COUNT(*) DESC", duration)
		if rows != nil {
			defer rows.Close()
			var byType []map[string]interface{}
			for rows.Next() {
				var et string
				var c int
				var as float64
				rows.Scan(&et, &c, &as)
				byType = append(byType, map[string]interface{}{"event_type": et, "count": c, "avg_score": as})
			}
			report["by_event_type"] = byType
		}

		// 按严重程度统计
		rows2, _ := db.QueryContext(r.Context(),
			"SELECT severity, COUNT(*) FROM risk_events WHERE detected_at > NOW() - $1::interval GROUP BY severity", duration)
		if rows2 != nil {
			defer rows2.Close()
			bySev := map[string]int{}
			for rows2.Next() {
				var s string
				var c int
				rows2.Scan(&s, &c)
				bySev[s] = c
			}
			report["by_severity"] = bySev
		}

		// 设置 JSON 下载头
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment; filename=risk-report-%s-%s.json", timeRange, time.Now().Format("20060102")))
		json.NewEncoder(w).Encode(report)
	}
}

// ==================== 审计日志 API ====================

func listAuditLogs(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		search := r.URL.Query().Get("search")
		page := parseIntDefault(r.URL.Query().Get("page"), 1)
		pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 20)

		result, err := svc.ListAuditLogs(r.Context(), action, search, page, pageSize)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: result})
	}
}

func getAuditActions(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actions, err := svc.GetAuditActions(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: actions})
	}
}

// ==================== 交易浏览器 API ====================

func getTransactionByHash(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := mux.Vars(r)["hash"]

		var tx models.ExplorerTransaction
		var callStack, eventsData, inputData, funcSel sql.NullString
		var gasLimit sql.NullInt64

		err := db.QueryRowContext(r.Context(),
			`SELECT t.tx_hash, t.block_number, t.from_address, COALESCE(t.to_address,''),
			        COALESCE(t.value::text,'0'), COALESCE(t.gas_price,0), COALESCE(t.gas_used,0),
			        t.gas_limit, t.input_data, t.function_selector, t.status, t.timestamp,
			        t.call_stack::text, t.events_data::text
			 FROM transactions t WHERE t.tx_hash = $1`, hash,
		).Scan(&tx.TxHash, &tx.BlockNumber, &tx.FromAddress, &tx.ToAddress,
			&tx.Value, &tx.GasPrice, &tx.GasUsed,
			&gasLimit, &inputData, &funcSel, &tx.Status, &tx.Timestamp,
			&callStack, &eventsData)
		if err != nil {
			respondError(w, http.StatusNotFound, "交易未找到")
			return
		}

		if gasLimit.Valid {
			tx.GasLimit = &gasLimit.Int64
		}
		if inputData.Valid && inputData.String != "" {
			tx.InputData = &inputData.String
		}
		if funcSel.Valid && funcSel.String != "" {
			tx.FunctionSelector = &funcSel.String
			// 查询函数名称
			var funcName, funcDesc sql.NullString
			db.QueryRowContext(r.Context(),
				"SELECT name, description FROM function_signatures WHERE selector = $1 LIMIT 1",
				funcSel.String).Scan(&funcName, &funcDesc)
			if funcName.Valid {
				tx.FunctionName = &funcName.String
			}
			if funcDesc.Valid {
				tx.FunctionDesc = &funcDesc.String
			}
		}
		if callStack.Valid && callStack.String != "" && callStack.String != "null" {
			tx.CallStack = json.RawMessage(callStack.String)
			// 为 call_stack 中每个 frame 解码函数名
			tx.CallStack = decodeCallStackFunctions(r.Context(), db, tx.CallStack)
		}
		if eventsData.Valid && eventsData.String != "" && eventsData.String != "null" {
			tx.EventsData = json.RawMessage(eventsData.String)
		}

		// 关联风险事件数
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM risk_events WHERE tx_hash = $1", hash).Scan(&tx.RiskCount)

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: tx})
	}
}

// decodeCallStackFunctions 给 call_stack 每个 frame 补充 function_name
func decodeCallStackFunctions(ctx context.Context, db *sql.DB, raw json.RawMessage) json.RawMessage {
	var frames []map[string]interface{}
	if err := json.Unmarshal(raw, &frames); err != nil {
		return raw
	}

	for i, frame := range frames {
		fn, ok := frame["function"]
		if !ok || fn == nil || fn == "" {
			continue
		}
		selector := fmt.Sprintf("%v", fn)
		if len(selector) < 10 {
			continue
		}
		var name, desc sql.NullString
		db.QueryRowContext(ctx,
			"SELECT name, description FROM function_signatures WHERE selector = $1 LIMIT 1",
			selector).Scan(&name, &desc)
		if name.Valid {
			frames[i]["function_name"] = name.String
		}
		if desc.Valid {
			frames[i]["function_desc"] = desc.String
		}
	}

	result, _ := json.Marshal(frames)
	return result
}

func getTransactionsByAddress(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := mux.Vars(r)["address"]
		page := parseIntDefault(r.URL.Query().Get("page"), 1)
		pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 20)
		if pageSize > 100 {
			pageSize = 100
		}

		// 总数
		var total int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM transactions WHERE from_address = $1 OR to_address = $1",
			address).Scan(&total)

		offset := (page - 1) * pageSize
		pages := total / pageSize
		if total%pageSize > 0 {
			pages++
		}

		rows, err := db.QueryContext(r.Context(),
			`SELECT tx_hash, block_number, from_address, COALESCE(to_address,''),
			        COALESCE(value::text,'0'), COALESCE(gas_used,0), function_selector, status, timestamp
			 FROM transactions WHERE from_address = $1 OR to_address = $1
			 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`,
			address, pageSize, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var txs []models.TxSummary
		for rows.Next() {
			var t models.TxSummary
			var funcSel sql.NullString
			if err := rows.Scan(&t.TxHash, &t.BlockNumber, &t.FromAddress, &t.ToAddress,
				&t.Value, &t.GasUsed, &funcSel, &t.Status, &t.Timestamp); err != nil {
				continue
			}
			if funcSel.Valid && funcSel.String != "" {
				t.FunctionSelector = &funcSel.String
				var name sql.NullString
				db.QueryRowContext(r.Context(),
					"SELECT name FROM function_signatures WHERE selector = $1 LIMIT 1",
					funcSel.String).Scan(&name)
				if name.Valid {
					t.FunctionName = &name.String
				}
			}
			txs = append(txs, t)
		}
		if txs == nil {
			txs = []models.TxSummary{}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items":     txs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
		})
	}
}

func getAddressSummary(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address := mux.Vars(r)["address"]
		summary := map[string]interface{}{"address": address}

		var txCount int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM transactions WHERE from_address = $1 OR to_address = $1",
			address).Scan(&txCount)
		summary["tx_count"] = txCount

		var sentCount, receivedCount int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM transactions WHERE from_address = $1", address).Scan(&sentCount)
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM transactions WHERE to_address = $1", address).Scan(&receivedCount)
		summary["sent_count"] = sentCount
		summary["received_count"] = receivedCount

		var riskCount int
		db.QueryRowContext(r.Context(),
			"SELECT COUNT(*) FROM risk_events WHERE contract_address = $1 OR tx_hash IN (SELECT tx_hash FROM transactions WHERE from_address = $1)",
			address).Scan(&riskCount)
		summary["risk_count"] = riskCount

		// 最常调用的函数
		funcRows, _ := db.QueryContext(r.Context(),
			`SELECT t.function_selector, COALESCE(f.name, t.function_selector), COUNT(*)
			 FROM transactions t
			 LEFT JOIN function_signatures f ON t.function_selector = f.selector
			 WHERE (t.from_address = $1 OR t.to_address = $1) AND t.function_selector IS NOT NULL AND t.function_selector != ''
			 GROUP BY t.function_selector, f.name
			 ORDER BY COUNT(*) DESC LIMIT 5`, address)
		if funcRows != nil {
			defer funcRows.Close()
			var topFuncs []map[string]interface{}
			for funcRows.Next() {
				var sel, name string
				var count int
				funcRows.Scan(&sel, &name, &count)
				topFuncs = append(topFuncs, map[string]interface{}{
					"selector": sel, "name": name, "count": count,
				})
			}
			summary["top_functions"] = topFuncs
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: summary})
	}
}

func getRisksByTxHash(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := mux.Vars(r)["hash"]

		rows, err := db.QueryContext(r.Context(),
			`SELECT id, event_type, severity, COALESCE(contract_address,''), tx_hash,
			        COALESCE(description,''), COALESCE(score,0), detected_at
			 FROM risk_events WHERE tx_hash = $1 ORDER BY detected_at`, hash)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var risks []models.RiskEvent
		for rows.Next() {
			var r models.RiskEvent
			if err := rows.Scan(&r.ID, &r.EventType, &r.Severity, &r.ContractAddress,
				&r.TxHash, &r.Description, &r.Score, &r.DetectedAt); err != nil {
				continue
			}
			risks = append(risks, r)
		}
		if risks == nil {
			risks = []models.RiskEvent{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: risks})
	}
}

func decodeFunctionSelector(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		selector := mux.Vars(r)["selector"]
		if !strings.HasPrefix(selector, "0x") {
			selector = "0x" + selector
		}

		rows, err := db.QueryContext(r.Context(),
			`SELECT selector, signature, name, COALESCE(category,''), COALESCE(description,''), is_privileged
			 FROM function_signatures WHERE selector = $1`, selector)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var sigs []models.FuncSig
		for rows.Next() {
			var s models.FuncSig
			if err := rows.Scan(&s.Selector, &s.Signature, &s.Name, &s.Category,
				&s.Description, &s.IsPrivileged); err != nil {
				continue
			}
			sigs = append(sigs, s)
		}
		if sigs == nil {
			sigs = []models.FuncSig{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: sigs})
	}
}

func getRecentTransactions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := parseIntDefault(r.URL.Query().Get("page"), 1)
		pageSize := parseIntDefault(r.URL.Query().Get("page_size"), 20)
		if pageSize > 100 {
			pageSize = 100
		}

		// 兼容旧的 limit 参数
		if r.URL.Query().Get("limit") != "" && r.URL.Query().Get("page") == "" {
			pageSize = parseIntDefault(r.URL.Query().Get("limit"), 20)
		}

		// 总数
		var total int
		db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM transactions").Scan(&total)

		offset := (page - 1) * pageSize
		pages := total / pageSize
		if total%pageSize > 0 {
			pages++
		}

		rows, err := db.QueryContext(r.Context(),
			`SELECT tx_hash, block_number, from_address, COALESCE(to_address,''),
			        COALESCE(value::text,'0'), COALESCE(gas_used,0), function_selector, status, timestamp
			 FROM transactions ORDER BY timestamp DESC LIMIT $1 OFFSET $2`, pageSize, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var txs []models.TxBrief
		for rows.Next() {
			var t models.TxBrief
			var funcSel sql.NullString
			if err := rows.Scan(&t.TxHash, &t.BlockNumber, &t.FromAddress, &t.ToAddress,
				&t.Value, &t.GasUsed, &funcSel, &t.Status, &t.Timestamp); err != nil {
				continue
			}
			if funcSel.Valid && funcSel.String != "" {
				t.FunctionSelector = &funcSel.String
				var name sql.NullString
				db.QueryRowContext(r.Context(),
					"SELECT name FROM function_signatures WHERE selector = $1 LIMIT 1",
					funcSel.String).Scan(&name)
				if name.Valid {
					t.FunctionName = &name.String
				}
			}
			txs = append(txs, t)
		}
		if txs == nil {
			txs = []models.TxBrief{}
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"items":     txs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pages":     pages,
		})
	}
}

func getAlertsByTxHash(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := mux.Vars(r)["hash"]

		rows, err := db.QueryContext(r.Context(),
			`SELECT a.id, a.risk_event_id, a.title, COALESCE(a.message,''), a.severity, a.status,
			        a.assigned_to, a.acknowledged_at, a.acknowledged_by, a.resolved_at, a.resolved_by,
			        a.notes, a.created_at,
			        COALESCE(r.tx_hash,''), COALESCE(r.contract_address,''), COALESCE(r.score,0)
			 FROM alerts a
			 JOIN risk_events r ON a.risk_event_id = r.id
			 WHERE r.tx_hash = $1
			 ORDER BY a.created_at DESC`, hash)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var alerts []models.Alert
		for rows.Next() {
			var a models.Alert
			if err := rows.Scan(&a.ID, &a.RiskEventID, &a.Title, &a.Message, &a.Severity, &a.Status,
				&a.AssignedTo, &a.AcknowledgedAt, &a.AcknowledgedBy, &a.ResolvedAt, &a.ResolvedBy,
				&a.Notes, &a.CreatedAt,
				&a.TxHash, &a.ContractAddress, &a.Score); err != nil {
				continue
			}
			alerts = append(alerts, a)
		}
		if alerts == nil {
			alerts = []models.Alert{}
		}

		respondJSON(w, http.StatusOK, APIResponse{Success: true, Data: alerts})
	}
}

// ==================== Helpers ====================

func parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if v, err := strconv.Atoi(s); err == nil && v > 0 {
		return v
	}
	return defaultVal
}

func rangeToDuration(r string) string {
	switch r {
	case "24h":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	case "90d":
		return "90 days"
	default:
		return "7 days"
	}
}
