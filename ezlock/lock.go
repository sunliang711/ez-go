package ezlock

import (
	"context"
	"errors"
	"time"
)

// 错误定义
var (
	// ErrLockNotAcquired 无法获取锁
	ErrLockNotAcquired = errors.New("lock not acquired")
	// ErrLockNotHeld 未持有锁
	ErrLockNotHeld = errors.New("lock not held")
	// ErrLockRenewalFailed 锁续约失败
	ErrLockRenewalFailed = errors.New("lock renewal failed")
)

// Lock 分布式锁接口
type Lock interface {
	// Acquire 获取锁
	// key: 锁的键
	// value: 锁的唯一标识值(用于识别锁的所有者)
	// ttl: 锁的过期时间
	// 如果获取锁失败,返回ErrLockNotAcquired
	Acquire(ctx context.Context, key string, value string, ttl time.Duration) error

	// Release 释放锁
	// key: 锁的键
	// value: 锁的唯一标识值(用于验证是否为锁的所有者)
	// 如果未持有锁,返回ErrLockNotHeld
	Release(ctx context.Context, key string, value string) error

	// Refresh 刷新锁的过期时间
	// key: 锁的键
	// value: 锁的唯一标识值(用于验证是否为锁的所有者)
	// ttl: 新的过期时间
	// 如果刷新失败,返回ErrLockRenewalFailed
	Refresh(ctx context.Context, key string, value string, ttl time.Duration) error

	// IsLocked 检查锁是否存在
	// key: 锁的键
	// 返回锁是否存在及锁的当前值(如果存在)
	IsLocked(ctx context.Context, key string) (bool, string, error)
}

// Option 锁配置选项
type Option func(*Options)

// Options 锁配置
type Options struct {
	// Redis选项
	RedisAddr      string // Redis地址
	RedisPassword  string // Redis密码
	RedisDB        int    // Redis数据库
	RedisKeyPrefix string // Redis键前缀

	// 重试选项
	RetryCount    int           // 获取锁的重试次数
	RetryInterval time.Duration // 重试间隔

	// 日志选项
	EnableLog bool // 是否启用日志
}

// WithRedisOptions 设置Redis选项
func WithRedisOptions(addr, password string, db int) Option {
	return func(o *Options) {
		o.RedisAddr = addr
		o.RedisPassword = password
		o.RedisDB = db
	}
}

// WithRedisKeyPrefix 设置Redis键前缀
func WithRedisKeyPrefix(prefix string) Option {
	return func(o *Options) {
		o.RedisKeyPrefix = prefix
	}
}

// WithRetryOptions 设置重试选项
func WithRetryOptions(retryCount int, retryInterval time.Duration) Option {
	return func(o *Options) {
		o.RetryCount = retryCount
		o.RetryInterval = retryInterval
	}
}

// WithEnableLog 设置是否启用日志
func WithEnableLog(enable bool) Option {
	return func(o *Options) {
		o.EnableLog = enable
	}
}

// NewRedisLock 创建一个基于Redis的分布式锁
func NewRedisLock(opts ...Option) (Lock, error) {
	options := &Options{
		RedisAddr:      "localhost:6379",
		RedisPassword:  "",
		RedisDB:        0,
		RedisKeyPrefix: "ezlock",
		RetryCount:     3,
		RetryInterval:  time.Millisecond * 100,
		EnableLog:      false,
	}

	for _, opt := range opts {
		opt(options)
	}

	return newRedisLock(options)
}
