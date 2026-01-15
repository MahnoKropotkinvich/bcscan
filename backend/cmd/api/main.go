package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/haswell/bcscan/internal/cache"
	"github.com/haswell/bcscan/internal/repository"
	"github.com/haswell/bcscan/internal/ruleengine"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Config struct {
	DatabaseURL string
	Port        string
	RedisAddr   string
	RulesPath   string
}

func loadConfig() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://bcscan:bcscan_password@postgres:5432/bcscan?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RulesPath:   getEnv("RULES_PATH", "./rules/builtin"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	// Initialize Redis, Repository and RuleManager
	redis := cache.NewRedisClient(cfg.RedisAddr)
	riskRepo := repository.NewRiskEventRepository(db, redis, logger)
	ruleManager := ruleengine.NewRuleManager(cfg.RulesPath, redis, logger)

	logger.Info("API Gateway starting", zap.String("port", cfg.Port))

	r := mux.NewRouter()
	r.Use(corsMiddleware)

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/risks", getRiskEvents(riskRepo)).Methods("GET")
	api.HandleFunc("/risks/{id}", getRiskEvent(riskRepo)).Methods("GET")
	api.HandleFunc("/stats", getStats(riskRepo)).Methods("GET")

	// Rule management routes
	api.HandleFunc("/rules", getRules(ruleManager)).Methods("GET")
	api.HandleFunc("/rules/reload", reloadRules(ruleManager)).Methods("POST")

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}

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

func getRiskEvents(repo *repository.RiskEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		severity := r.URL.Query().Get("severity")
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil {
				limit = parsed
			}
		}

		events, err := repo.List(r.Context(), severity, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	}
}

func getRiskEvent(repo *repository.RiskEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		event, err := repo.GetByID(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	}
}

func getStats(repo *repository.RiskEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := repo.GetStats(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func getRules(rm *ruleengine.RuleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules := rm.GetRules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rules)
	}
}

func reloadRules(rm *ruleengine.RuleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		if err := rm.LoadRules(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := rm.PublishUpdate(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"count":   len(rm.GetRules()),
		})
	}
}
