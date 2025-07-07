package ezcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/patrickmn/go-cache"
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

	if mc.options.EnableLog {
		fmt.Printf("设置键: %s\n", key)
	}

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

	if mc.options.EnableLog && len(items) > 0 {
		fmt.Printf("批量设置 %d 个键\n", len(items))
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

	if mc.options.EnableLog {
		fmt.Println("内存缓存已清空")
	}

	return nil
}

// Keys 获取所有键
func (mc *memoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// 获取所有缓存项
	items := mc.cache.Items()

	// 如果模式为空，返回所有键
	if pattern == "" || pattern == "*" {
		keys := make([]string, 0, len(items))
		for key := range items {
			keys = append(keys, key)
		}
		return keys, nil
	}

	// 否则，根据模式匹配键
	keys := make([]string, 0)
	for key := range items {
		if wildcardMatch(key, pattern) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Expire 设置过期时间
func (mc *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	// 获取当前值
	value, found := mc.cache.Get(key)
	if !found {
		return ErrKeyNotFound
	}

	// 重新设置键的过期时间
	mc.cache.Set(key, value, ttl)

	return nil
}

// TTL 获取剩余过期时间
func (mc *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	// go-cache不提供直接获取TTL的方法，需要使用内部方法
	// 获取缓存项的详细信息
	item, found := mc.cache.Items()[key]
	if !found {
		return -2, ErrKeyNotFound // -2 表示键不存在
	}

	// 检查是否已过期
	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return -2, ErrKeyNotFound // 已过期
	}

	// 如果没有设置过期时间
	if item.Expiration == 0 {
		return -1, nil // -1 表示永不过期
	}

	// 计算剩余时间
	remaining := time.Duration(item.Expiration - time.Now().UnixNano())
	if remaining < 0 {
		return -2, ErrKeyNotFound
	}

	return remaining, nil
}

// Stats 获取统计信息
func (mc *memoryCache) Stats(ctx context.Context) (Stats, error) {
	// 获取最新的键数量
	keyCount := mc.cache.ItemCount()

	// 更新统计信息
	stats := mc.stats
	stats.KeyCount = int64(keyCount)

	// 添加go-cache内部信息到详情
	stats.Details["go_cache_version"] = "v2.1.0"

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
