package ezcache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// redisCache Redis缓存实现
type redisCache struct {
	client  *redis.Client
	options *Options
	// 统计计数器
	hitCount    int64
	missCount   int64
	evictCount  int64
	expireCount int64
}

// newRedisCache 创建一个新的Redis缓存
func newRedisCache(options *Options) (Cache, error) {
	if options.RedisAddr == "" {
		return nil, fmt.Errorf("redis address is required")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     options.RedisAddr,
		Password: options.RedisPassword,
		DB:       options.RedisDB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("connect to redis failed: %w", err)
	}

	rc := &redisCache{
		client:  client,
		options: options,
	}

	return rc, nil
}

// buildKey 构建Redis键，添加前缀
func (rc *redisCache) buildKey(key string) string {
	if rc.options.RedisKeyPrefix == "" {
		return key
	}
	return rc.options.RedisKeyPrefix + ":" + key
}

// Set 设置缓存
func (rc *redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	// 如果未指定TTL，使用默认值
	if ttl == 0 {
		ttl = rc.options.DefaultTTL
	}

	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// 构建键
	redisKey := rc.buildKey(key)

	// 设置键值
	err = rc.client.Set(ctx, redisKey, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set value in redis: %w", err)
	}

	Log(rc.options.EnableLog, zerolog.InfoLevel, "Key set in redis cache: %s", key)

	return nil
}

// Get 获取缓存
func (rc *redisCache) Get(ctx context.Context, key string, valuePtr interface{}) error {
	redisKey := rc.buildKey(key)

	// 获取值
	data, err := rc.client.Get(ctx, redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&rc.missCount, 1)
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to get value from redis: %w", err)
	}

	// 反序列化值
	err = json.Unmarshal(data, valuePtr)
	if err != nil {
		atomic.AddInt64(&rc.missCount, 1)
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	atomic.AddInt64(&rc.hitCount, 1)
	return nil
}

// Delete 删除缓存
func (rc *redisCache) Delete(ctx context.Context, key string) error {
	redisKey := rc.buildKey(key)

	// 删除键
	err := rc.client.Del(ctx, redisKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key from redis: %w", err)
	}

	Log(rc.options.EnableLog, zerolog.InfoLevel, "Key deleted from redis cache: %s", key)

	return nil
}

// Exists 检查键是否存在
func (rc *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	redisKey := rc.buildKey(key)

	// 检查键是否存在
	exists, err := rc.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if key exists in redis: %w", err)
	}

	return exists > 0, nil
}

// MSet 批量设置
func (rc *redisCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	if len(items) == 0 {
		return nil
	}

	// 如果未指定TTL，使用默认值
	if ttl == 0 {
		ttl = rc.options.DefaultTTL
	}

	// 批量设置
	pipe := rc.client.Pipeline()
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}

		redisKey := rc.buildKey(key)
		pipe.Set(ctx, redisKey, data, ttl)
	}

	// 执行批量命令
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	Log(rc.options.EnableLog, zerolog.InfoLevel, "Batch set %d keys in redis cache", len(items))

	return nil
}

// MGet 批量获取
func (rc *redisCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// 转换键
	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = rc.buildKey(key)
	}

	// 批量获取
	values, err := rc.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to mget from redis: %w", err)
	}

	result := make(map[string]interface{})
	for i, value := range values {
		if value == nil {
			atomic.AddInt64(&rc.missCount, 1)
			continue
		}

		// 将键转换回原始键
		originalKey := keys[i]

		// 反序列化值
		var decodedValue interface{}
		err := json.Unmarshal([]byte(value.(string)), &decodedValue)
		if err != nil {
			atomic.AddInt64(&rc.missCount, 1)
			continue
		}

		result[originalKey] = decodedValue
		atomic.AddInt64(&rc.hitCount, 1)
	}

	return result, nil
}

// Clear 清空与前缀匹配的所有键
func (rc *redisCache) Clear(ctx context.Context) error {
	pattern := rc.buildKey("*") // 所有键

	// 查找匹配的键
	iter := rc.client.Scan(ctx, 0, pattern, 0).Iterator()

	// 删除匹配的键
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())

		// 批量删除，每100个键执行一次
		if len(keys) >= 100 {
			err := rc.client.Del(ctx, keys...).Err()
			if err != nil {
				return fmt.Errorf("failed to clear keys from redis: %w", err)
			}
			atomic.AddInt64(&rc.evictCount, int64(len(keys)))
			keys = keys[:0]
		}
	}

	// 删除剩余的键
	if len(keys) > 0 {
		err := rc.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to clear keys from redis: %w", err)
		}
		atomic.AddInt64(&rc.evictCount, int64(len(keys)))
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate over keys: %w", err)
	}

	Log(rc.options.EnableLog, zerolog.InfoLevel, "Redis cache cleared")

	return nil
}

// Keys 获取所有键
func (rc *redisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// 构建搜索模式
	var searchPattern string
	if pattern == "" {
		searchPattern = rc.buildKey("*")
	} else {
		// 将模式中的*和?保留作为Redis通配符
		searchPattern = rc.buildKey(pattern)
	}

	// 查找匹配的键
	var keys []string
	iter := rc.client.Scan(ctx, 0, searchPattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		// 如果有前缀，移除前缀
		if rc.options.RedisKeyPrefix != "" {
			key = strings.TrimPrefix(key, rc.options.RedisKeyPrefix+":")
		}

		keys = append(keys, key)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over keys: %w", err)
	}

	return keys, nil
}

// Expire 设置过期时间
func (rc *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	redisKey := rc.buildKey(key)

	// 检查键是否存在
	exists, err := rc.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check if key exists in redis: %w", err)
	}

	if exists == 0 {
		return ErrKeyNotFound
	}

	// 设置过期时间
	success, err := rc.client.Expire(ctx, redisKey, ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to set expiration in redis: %w", err)
	}

	if !success {
		return ErrKeyNotFound
	}

	return nil
}

// TTL 获取剩余过期时间
func (rc *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	redisKey := rc.buildKey(key)

	// 获取TTL
	ttl, err := rc.client.TTL(ctx, redisKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL from redis: %w", err)
	}

	// Redis返回的特殊值处理
	if ttl == -2 {
		return -2, ErrKeyNotFound // 键不存在
	}

	return ttl, nil
}

// Stats 获取统计信息
func (rc *redisCache) Stats(ctx context.Context) (Stats, error) {
	// 获取Redis统计信息
	info, err := rc.client.Info(ctx, "keyspace").Result()
	if err != nil {
		return Stats{}, fmt.Errorf("failed to get redis stats: %w", err)
	}

	stats := Stats{
		HitCount:     atomic.LoadInt64(&rc.hitCount),
		MissCount:    atomic.LoadInt64(&rc.missCount),
		EvictedCount: atomic.LoadInt64(&rc.evictCount),
		ExpiredCount: atomic.LoadInt64(&rc.expireCount),
		Details:      make(map[string]interface{}),
	}

	// 统计键数量
	pattern := rc.buildKey("*")
	keyCount, err := rc.client.Keys(ctx, pattern).Result()
	if err != nil {
		return stats, nil // 返回基本统计信息
	}
	stats.KeyCount = int64(len(keyCount))

	// 添加Redis信息
	stats.Details["redis_info"] = info

	return stats, nil
}
