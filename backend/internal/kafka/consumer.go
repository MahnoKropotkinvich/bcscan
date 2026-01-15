package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer Kafka 消费者
type Consumer struct {
	reader *kafka.Reader
	logger *zap.Logger
}

// NewConsumer 创建新的消费者
func NewConsumer(brokers []string, topic string, groupID string, logger *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}
}

// ReadMessage 读取消息
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

// Unmarshal 反序列化消息
func (c *Consumer) Unmarshal(msg kafka.Message, v interface{}) error {
	return json.Unmarshal(msg.Value, v)
}

// Close 关闭消费者
func (c *Consumer) Close() error {
	return c.reader.Close()
}
