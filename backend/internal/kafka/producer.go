package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Producer Kafka 生产者
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

// NewProducer 创建新的生产者
func NewProducer(brokers []string, topic string, logger *zap.Logger) *Producer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

// SendMessage 发送消息
func (p *Producer) SendMessage(ctx context.Context, key string, value interface{}) error {
	// 序列化消息
	data, err := json.Marshal(value)
	if err != nil {
		p.logger.Error("Failed to marshal message", zap.Error(err))
		return err
	}

	// 发送消息
	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("Failed to write message", zap.Error(err))
		return err
	}

	return nil
}

// Close 关闭生产者
func (p *Producer) Close() error {
	return p.writer.Close()
}
