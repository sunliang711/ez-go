package ezcache

import (
	"context"
	"errors"
	"time"
)

// 通用错误定义
var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrKeyExists        = errors.New("key already exists")
	ErrCacheNotInit     = errors.New("cache not initialized")
	ErrInvalidTTL       = errors.New("invalid TTL")
	ErrInvalidCacheType = errors.New("invalid cache type")
)

// Cache 定义缓存接口
type Cache interface {
	// 基本操作
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 批量操作
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	MGet(ctx context.Context, keys []string) (map[string]interface{}, error)

	// 辅助操作
	Clear(ctx context.Context) error
	Keys(ctx context.Context, pattern string) ([]string, error)

	// TTL相关
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 统计相关
	Stats(ctx context.Context) (Stats, error)
}

// Stats 缓存统计信息
type Stats struct {
	KeyCount     int64                  // 键数量
	HitCount     int64                  // 命中次数
	MissCount    int64                  // 未命中次数
	EvictedCount int64                  // 被清除数量
	ExpiredCount int64                  // 过期数量
	Details      map[string]interface{} // 额外详细信息
}

// CacheType 缓存类型
type CacheType int

const (
	MemoryCache CacheType = iota // 内存缓存
	RedisCache                   // Redis缓存
)

// Option 缓存选项
type Option func(*Options)

// Options 缓存配置选项
type Options struct {
	CacheType       CacheType     // 缓存类型
	DefaultTTL      time.Duration // 默认过期时间
	CleanupInterval time.Duration // 清理间隔(内存缓存)
	MaxEntries      int           // 最大条目数(内存缓存)

	// Redis选项
	RedisAddr      string // Redis地址
	RedisUsername  string // Redis用户名
	RedisPassword  string // Redis密码
	RedisDB        int    // Redis数据库
	RedisKeyPrefix string // Redis键前缀

	// 日志选项
	EnableLog bool // 是否启用日志
}

// WithCacheType 设置缓存类型
func WithCacheType(t CacheType) Option {
	return func(o *Options) {
		o.CacheType = t
	}
}

// WithDefaultTTL 设置默认过期时间
func WithDefaultTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.DefaultTTL = ttl
	}
}

// WithCleanupInterval 设置清理间隔
func WithCleanupInterval(interval time.Duration) Option {
	return func(o *Options) {
		o.CleanupInterval = interval
	}
}

// WithMaxEntries 设置最大条目数
func WithMaxEntries(max int) Option {
	return func(o *Options) {
		o.MaxEntries = max
	}
}

// WithRedisOptions 设置Redis选项
func WithRedisOptions(addr, username, password string, db int) Option {
	return func(o *Options) {
		o.RedisAddr = addr
		o.RedisPassword = password
		o.RedisDB = db
		o.RedisUsername = username
	}
}

// WithRedisKeyPrefix 设置Redis键前缀
func WithRedisKeyPrefix(prefix string) Option {
	return func(o *Options) {
		o.RedisKeyPrefix = prefix
	}
}

// WithEnableLog 设置是否启用日志
func WithEnableLog(enable bool) Option {
	return func(o *Options) {
		o.EnableLog = enable
	}
}

// NewCache 创建一个新的缓存实例
func NewCache(opts ...Option) (Cache, error) {
	options := &Options{
		CacheType:       MemoryCache,
		DefaultTTL:      time.Hour,
		CleanupInterval: time.Minute * 5,
		MaxEntries:      10000,
		EnableLog:       false,
	}

	for _, opt := range opts {
		opt(options)
	}

	switch options.CacheType {
	case MemoryCache:
		return newMemoryCache(options)
	case RedisCache:
		return newRedisCache(options)
	default:
		return nil, ErrInvalidCacheType
	}
}
