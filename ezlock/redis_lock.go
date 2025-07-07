package ezlock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// redisLock Redis分布式锁实现
type redisLock struct {
	client  *redis.Client
	options *Options
}

// newRedisLock 创建一个Redis分布式锁
func newRedisLock(options *Options) (Lock, error) {
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

	rl := &redisLock{
		client:  client,
		options: options,
	}

	return rl, nil
}

// buildKey 构建Redis键，添加前缀
func (rl *redisLock) buildKey(key string) string {
	if rl.options.RedisKeyPrefix == "" {
		return key
	}
	return rl.options.RedisKeyPrefix + ":" + key
}

// Acquire 获取锁
func (rl *redisLock) Acquire(ctx context.Context, key string, value string, ttl time.Duration) error {
	redisKey := rl.buildKey(key)

	// 使用SET命令的NX选项实现分布式锁
	// NX: 仅当键不存在时才设置
	// PX: 以毫秒为单位设置过期时间
	success, err := rl.client.SetNX(ctx, redisKey, value, ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !success {
		// 如果配置了重试选项，进行重试
		if rl.options.RetryCount > 0 {
			return rl.retryAcquire(ctx, key, value, ttl)
		}
		return ErrLockNotAcquired
	}

	Log(rl.options.EnableLog, zerolog.InfoLevel, "Lock acquired: %s (value: %s, ttl: %v)", key, value, ttl)

	return nil
}

// retryAcquire 重试获取锁
func (rl *redisLock) retryAcquire(ctx context.Context, key string, value string, ttl time.Duration) error {
	redisKey := rl.buildKey(key)

	for i := 0; i < rl.options.RetryCount; i++ {
		// 等待重试间隔
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rl.options.RetryInterval):
			// 继续重试
		}

		// 重试获取锁
		success, err := rl.client.SetNX(ctx, redisKey, value, ttl).Result()
		if err != nil {
			return fmt.Errorf("failed to acquire lock (retry %d): %w", i+1, err)
		}

		if success {
			Log(rl.options.EnableLog, zerolog.InfoLevel, "Lock acquired after %d retries: %s (value: %s, ttl: %v)", i+1, key, value, ttl)
			return nil
		}

		Log(rl.options.EnableLog, zerolog.DebugLevel, "Lock acquisition failed (retry %d): %s", i+1, key)
	}

	return ErrLockNotAcquired
}

// Release 释放锁
func (rl *redisLock) Release(ctx context.Context, key string, value string) error {
	redisKey := rl.buildKey(key)

	// 使用Lua脚本确保只有锁的拥有者可以释放锁
	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`

	result, err := rl.client.Eval(ctx, script, []string{redisKey}, value).Int()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	Log(rl.options.EnableLog, zerolog.InfoLevel, "Lock released: %s (value: %s)", key, value)

	return nil
}

// Refresh 刷新锁的过期时间
func (rl *redisLock) Refresh(ctx context.Context, key string, value string, ttl time.Duration) error {
	redisKey := rl.buildKey(key)

	// 使用Lua脚本确保只有锁的拥有者可以刷新锁
	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("PEXPIRE", KEYS[1], ARGV[2])
	else
		return 0
	end
	`

	result, err := rl.client.Eval(ctx, script, []string{redisKey}, value, ttl.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	if result == 0 {
		return ErrLockRenewalFailed
	}

	Log(rl.options.EnableLog, zerolog.InfoLevel, "Lock refreshed: %s (value: %s, ttl: %v)", key, value, ttl)

	return nil
}

// IsLocked 检查锁是否存在
func (rl *redisLock) IsLocked(ctx context.Context, key string) (bool, string, error) {
	redisKey := rl.buildKey(key)

	value, err := rl.client.Get(ctx, redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			// 键不存在，锁未被获取
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check lock: %w", err)
	}

	// 键存在，锁已被获取
	return true, value, nil
}
