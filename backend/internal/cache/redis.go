package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr string) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 3,
	})

	return &RedisClient{client: rdb}
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) GetRaw(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Publish 发布消息到 Redis 频道
// message 可以是 string（直接发送）或其他类型（JSON 序列化后发送）
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	var payload string
	switch v := message.(type) {
	case string:
		payload = v
	default:
		data, err := json.Marshal(message)
		if err != nil {
			return err
		}
		payload = string(data)
	}
	return r.client.Publish(ctx, channel, payload).Err()
}

func (r *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return r.client.Subscribe(ctx, channel)
}

func (r *RedisClient) Incr(ctx context.Context, key string) error {
	return r.client.Incr(ctx, key).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
