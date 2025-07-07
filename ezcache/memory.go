package ezcache

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// memoryCache 是基于go-cache的内存缓存实现
type memoryCache struct {
	cache   *cache.Cache
	options *Options
	stats   Stats
}

// newMemoryCache 创建一个新的内存缓存
func newMemoryCache(options *Options) (Cache, error) {
	// 设置默认清理间隔和默认过期时间
	cleanupInterval := options.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute * 5
	}

	defaultTTL := options.DefaultTTL
	if defaultTTL <= 0 {
		defaultTTL = time.Hour
	}

	mc := &memoryCache{
		cache:   cache.New(defaultTTL, cleanupInterval),
		options: options,
		stats: Stats{
			Details: make(map[string]interface{}),
		},
	}

	return mc, nil
}

// Set 设置缓存
func (mc *memoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	// 使用指定的TTL或默认TTL
	if ttl == 0 {
		ttl = mc.options.DefaultTTL
	}

	// 设置缓存
	mc.cache.Set(key, value, ttl)

	Log(mc.options.EnableLog, zerolog.InfoLevel, "设置键: %s", key)

	return nil
}

// Get 获取缓存
func (mc *memoryCache) Get(ctx context.Context, key string, valuePtr interface{}) error {
	// 获取值
	value, found := mc.cache.Get(key)
	if !found {
		mc.stats.MissCount++
		return ErrKeyNotFound
	}

	// 尝试将缓存的值转换为目标类型
	err := mc.convertValue(value, valuePtr)
	if err != nil {
		mc.stats.MissCount++
		return err
	}

	mc.stats.HitCount++
	return nil
}

// convertValue 将缓存值转换为目标类型
func (mc *memoryCache) convertValue(src, dst interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}

	// 如果源和目标类型相同，直接赋值
	srcValue := reflect.ValueOf(src)
	if srcValue.Type().AssignableTo(dstValue.Elem().Type()) {
		dstValue.Elem().Set(srcValue)
		return nil
	}

	// 尝试通过JSON序列化/反序列化进行转换
	jsonData, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, dst)
}

// Delete 删除缓存
func (mc *memoryCache) Delete(ctx context.Context, key string) error {
	mc.cache.Delete(key)
	Log(mc.options.EnableLog, zerolog.InfoLevel, "删除键: %s", key)
	return nil
}

// Exists 检查键是否存在
func (mc *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	_, found := mc.cache.Get(key)
	return found, nil
}

// MSet 批量设置
func (mc *memoryCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	// 使用指定的TTL或默认TTL
	if ttl == 0 {
		ttl = mc.options.DefaultTTL
	}

	// 批量设置
	for key, value := range items {
		mc.cache.Set(key, value, ttl)
	}

	if len(items) > 0 {
		Log(mc.options.EnableLog, zerolog.InfoLevel, "批量设置 %d 个键", len(items))
	}

	return nil
}

// MGet 批量获取
func (mc *memoryCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, key := range keys {
		value, found := mc.cache.Get(key)
		if !found {
			mc.stats.MissCount++
			continue
		}

		result[key] = value
		mc.stats.HitCount++
	}

	return result, nil
}

// Clear 清空缓存
func (mc *memoryCache) Clear(ctx context.Context) error {
	// 记录被清除的键数量
	oldCount := mc.cache.ItemCount()
	mc.stats.EvictedCount += int64(oldCount)

	// 清空缓存
	mc.cache.Flush()

	Log(mc.options.EnableLog, zerolog.InfoLevel, "内存缓存已清空")

	return nil
}

// Keys 获取所有键
func (mc *memoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// 获取所有键
	items := mc.cache.Items()
	keys := make([]string, 0, len(items))

	// 如果没有指定模式，返回所有键
	if pattern == "" {
		for key := range items {
			keys = append(keys, key)
		}
		return keys, nil
	}

	// TODO: 实现模式匹配功能
	// 目前简单返回所有键
	for key := range items {
		keys = append(keys, key)
	}

	return keys, nil
}

// Expire 设置过期时间
func (mc *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	// 检查键是否存在
	item, found := mc.cache.Get(key)
	if !found {
		return ErrKeyNotFound
	}

	// 重新设置带新TTL的键
	mc.cache.Set(key, item, ttl)

	Log(mc.options.EnableLog, zerolog.InfoLevel, "键过期时间已更新: %s, ttl: %v", key, ttl)

	return nil
}

// TTL 获取剩余过期时间
func (mc *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	// 检查键是否存在
	_, expires, found := mc.cache.GetWithExpiration(key)
	if !found {
		return -2, ErrKeyNotFound // 与Redis TTL返回值一致：-2表示键不存在
	}

	// 计算剩余时间
	remaining := time.Until(expires)
	if remaining < 0 {
		// 已过期，会在清理时删除
		return -1, nil // 与Redis TTL返回值一致：-1表示键已过期
	}

	return remaining, nil
}

// Stats 获取统计信息
func (mc *memoryCache) Stats(ctx context.Context) (Stats, error) {
	stats := mc.stats
	stats.KeyCount = int64(mc.cache.ItemCount())

	return stats, nil
}

// wildcardMatch 实现简单的通配符匹配
// pattern中可以使用*匹配任意字符，?匹配单个字符
func wildcardMatch(s, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 这里我们实现一个简单的模式匹配
	// 实际生产中可能需要更复杂的实现或使用正则表达式
	if pattern == s {
		return true
	}

	// 检查是否是以*开头的模式
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(s) >= len(prefix) && s[:len(prefix)] == prefix
	}

	// 检查是否是以*结尾的模式
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
	}

	return false // 其他情况默认不匹配
}
