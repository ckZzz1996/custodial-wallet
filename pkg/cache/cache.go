package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"custodial-wallet/pkg/config"
	"custodial-wallet/pkg/logger"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init 初始化Redis连接
func Init(cfg config.RedisConfig) error {
	client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis connected successfully")
	return nil
}

// GetClient 获取Redis客户端
func GetClient() *redis.Client {
	return client
}

// Close 关闭Redis连接
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// Set 设置缓存
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func Get(ctx context.Context, key string, dest interface{}) error {
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete 删除缓存
func Delete(ctx context.Context, keys ...string) error {
	return client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	result, err := client.Exists(ctx, key).Result()
	return result > 0, err
}

// SetNX 设置缓存（如果不存在）
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return client.SetNX(ctx, key, data, expiration).Result()
}

// Incr 递增
func Incr(ctx context.Context, key string) (int64, error) {
	return client.Incr(ctx, key).Result()
}

// Decr 递减
func Decr(ctx context.Context, key string) (int64, error) {
	return client.Decr(ctx, key).Result()
}

// TTL 获取过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return client.TTL(ctx, key).Result()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	return client.Expire(ctx, key, expiration).Err()
}

// Lock 分布式锁
type Lock struct {
	key    string
	value  string
	expiry time.Duration
}

// NewLock 创建分布式锁
func NewLock(key string, expiry time.Duration) *Lock {
	return &Lock{
		key:    "lock:" + key,
		value:  fmt.Sprintf("%d", time.Now().UnixNano()),
		expiry: expiry,
	}
}

// Acquire 获取锁
func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	return client.SetNX(ctx, l.key, l.value, l.expiry).Result()
}

// Release 释放锁
func (l *Lock) Release(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	return client.Eval(ctx, script, []string{l.key}, l.value).Err()
}
