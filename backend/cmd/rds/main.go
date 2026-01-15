package main

import (
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	// 初始化 logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("Risk Detection Service starting...")

	// 加载配置
	cfg := loadConfig()

	// 连接数据库
	db, err := connectDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connected successfully")

	// 创建并启动服务
	service := NewRDSService(db, cfg, logger)
	if err := service.Start(); err != nil {
		logger.Fatal("Failed to start service", zap.Error(err))
	}

	// 等待退出信号
	waitForShutdown(service, logger)
}

// Config RDS 配置
type Config struct {
	DatabaseURL string
	KafkaBroker string
	KafkaTopic  string
	RulesPath   string
}

// loadConfig 加载配置
func loadConfig() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://bcscan:bcscan123@localhost:5432/bcscan?sslmode=disable"),
		KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:  getEnv("KAFKA_TOPIC", "blockchain.transactions"),
		RulesPath:   getEnv("RULES_PATH", "./rules/builtin"),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// connectDatabase 连接数据库
func connectDatabase(cfg *Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// waitForShutdown 等待关闭信号
func waitForShutdown(service *RDSService, logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutdown signal received, stopping service...")

	service.Stop()
	logger.Info("Service stopped gracefully")
}
