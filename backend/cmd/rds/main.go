package main

import (
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	"github.com/haswell/bcscan/internal/config"
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
	RedisAddr   string
}

// loadConfig 加载配置
func loadConfig() *Config {
	return &Config{
		DatabaseURL: config.GetEnv("DATABASE_URL", "postgres://bcscan:bcscan123@localhost:5432/bcscan?sslmode=disable"),
		KafkaBroker: config.GetEnv("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:  config.GetEnv("KAFKA_TOPIC", "blockchain.transactions"),
		RulesPath:   config.GetEnv("RULES_PATH", "./rules/builtin"),
		RedisAddr:   config.GetEnv("REDIS_ADDR", "localhost:6379"),
	}
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
