package ezcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// memoryCache 是内存缓存实现
type memoryCache struct {
	data            sync.Map
	options         *Options
	stats           Stats
	stopCleanupChan chan struct{}
	statsMutex      sync.RWMutex
}

// cacheItem 缓存项结构
type cacheItem struct {
	Value      interface{}
	Expiration int64 // Unix时间戳，0表示不过期
}

// 判断是否过期
func (item *cacheItem) isExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// newMemoryCache 创建一个新的内存缓存
func newMemoryCache(options *Options) (Cache, error) {
	mc := &memoryCache{
		options: options,
		stats: Stats{
			Details: make(map[string]interface{}),
		},
		stopCleanupChan: make(chan struct{}),
	}

	// 启动定期清理过期项
	go mc.startCleanupTimer()

	return mc, nil
}

// startCleanupTimer 启动定期清理
func (mc *memoryCache) startCleanupTimer() {
	ticker := time.NewTicker(mc.options.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.cleanupExpired()
		case <-mc.stopCleanupChan:
			return
		}
	}
}

// cleanupExpired 清理过期项
func (mc *memoryCache) cleanupExpired() {
	var expiredCount int64

	mc.data.Range(func(key, value interface{}) bool {
		item, ok := value.(*cacheItem)
		if !ok {
			return true
		}

		if item.isExpired() {
			keyStr, _ := key.(string)
			mc.data.Delete(key)
			expiredCount++

			if mc.options.EnableLog {
				fmt.Printf("过期键已删除: %s\n", keyStr)
			}
		}
		return true
	})

	if expiredCount > 0 {
		mc.statsMutex.Lock()
		mc.stats.ExpiredCount += expiredCount
		mc.statsMutex.Unlock()

		if mc.options.EnableLog {
			fmt.Printf("清理了 %d 个过期键\n", expiredCount)
		}
	}
}

// Set 设置缓存
func (mc *memoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var expiration int64

	if ttl < 0 {
		return ErrInvalidTTL
	} else if ttl == 0 {
		// 使用默认TTL
		if mc.options.DefaultTTL > 0 {
			expiration = time.Now().Add(mc.options.DefaultTTL).UnixNano()
		}
	} else {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	mc.data.Store(key, &cacheItem{
		Value:      value,
		Expiration: expiration,
	})

	if mc.options.EnableLog {
		fmt.Printf("设置键: %s\n", key)
	}

	return nil
}

// Get 获取缓存
func (mc *memoryCache) Get(ctx context.Context, key string, valuePtr interface{}) error {
	value, ok := mc.data.Load(key)
	if !ok {
		mc.incrementMissCount()
		return ErrKeyNotFound
	}

	item, ok := value.(*cacheItem)
	if !ok {
		mc.incrementMissCount()
		return ErrKeyNotFound
	}

	if item.isExpired() {
		mc.data.Delete(key)
		mc.incrementMissCount()
		mc.incrementExpiredCount()
		return ErrKeyNotFound
	}

	// 尝试将缓存的值转换为目标类型
	err := mc.convertValue(item.Value, valuePtr)
	if err != nil {
		mc.incrementMissCount()
		return err
	}

	mc.incrementHitCount()
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
	mc.data.Delete(key)
	return nil
}

// Exists 检查键是否存在
func (mc *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	value, ok := mc.data.Load(key)
	if !ok {
		return false, nil
	}

	item, ok := value.(*cacheItem)
	if !ok {
		return false, nil
	}

	if item.isExpired() {
		mc.data.Delete(key)
		mc.incrementExpiredCount()
		return false, nil
	}

	return true, nil
}

// MSet 批量设置
func (mc *memoryCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	var expiration int64
	if ttl == 0 {
		// 使用默认TTL
		if mc.options.DefaultTTL > 0 {
			expiration = time.Now().Add(mc.options.DefaultTTL).UnixNano()
		}
	} else {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	for key, value := range items {
		mc.data.Store(key, &cacheItem{
			Value:      value,
			Expiration: expiration,
		})
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
		value, ok := mc.data.Load(key)
		if !ok {
			mc.incrementMissCount()
			continue
		}

		item, ok := value.(*cacheItem)
		if !ok {
			mc.incrementMissCount()
			continue
		}

		if item.isExpired() {
			mc.data.Delete(key)
			mc.incrementMissCount()
			mc.incrementExpiredCount()
			continue
		}

		result[key] = item.Value
		mc.incrementHitCount()
	}

	return result, nil
}

// Clear 清空缓存
func (mc *memoryCache) Clear(ctx context.Context) error {
	// 创建一个新的sync.Map替换掉旧的
	mc.data = sync.Map{}

	mc.statsMutex.Lock()
	defer mc.statsMutex.Unlock()

	// 重置统计信息
	mc.stats.KeyCount = 0
	mc.stats.EvictedCount += mc.stats.KeyCount

	if mc.options.EnableLog {
		fmt.Println("内存缓存已清空")
	}

	return nil
}

// Keys 获取所有键
func (mc *memoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string

	mc.data.Range(func(key, value interface{}) bool {
		keyStr, ok := key.(string)
		if !ok {
			return true
		}

		item, ok := value.(*cacheItem)
		if !ok {
			return true
		}

		if item.isExpired() {
			mc.data.Delete(key)
			mc.incrementExpiredCount()
			return true
		}

		// 如果pattern为空或者模式匹配
		if pattern == "" || wildcardMatch(keyStr, pattern) {
			keys = append(keys, keyStr)
		}

		return true
	})

	return keys, nil
}

// Expire 设置过期时间
func (mc *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl < 0 {
		return ErrInvalidTTL
	}

	valueIface, ok := mc.data.Load(key)
	if !ok {
		return ErrKeyNotFound
	}

	item, ok := valueIface.(*cacheItem)
	if !ok {
		return ErrKeyNotFound
	}

	if item.isExpired() {
		mc.data.Delete(key)
		mc.incrementExpiredCount()
		return ErrKeyNotFound
	}

	// 计算新的过期时间
	var expiration int64
	if ttl == 0 {
		expiration = 0 // 永不过期
	} else {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	// 创建新的item并存储
	mc.data.Store(key, &cacheItem{
		Value:      item.Value,
		Expiration: expiration,
	})

	return nil
}

// TTL 获取剩余过期时间
func (mc *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	valueIface, ok := mc.data.Load(key)
	if !ok {
		return -2, ErrKeyNotFound // -2 表示键不存在
	}

	item, ok := valueIface.(*cacheItem)
	if !ok {
		return -2, ErrKeyNotFound
	}

	if item.isExpired() {
		mc.data.Delete(key)
		mc.incrementExpiredCount()
		return -2, ErrKeyNotFound
	}

	// 如果没有设置过期时间
	if item.Expiration == 0 {
		return -1, nil // -1 表示永不过期
	}

	// 计算剩余时间
	remaining := time.Duration(item.Expiration - time.Now().UnixNano())
	if remaining < 0 {
		mc.data.Delete(key)
		mc.incrementExpiredCount()
		return -2, ErrKeyNotFound
	}

	return remaining, nil
}

// Stats 获取统计信息
func (mc *memoryCache) Stats(ctx context.Context) (Stats, error) {
	mc.statsMutex.RLock()
	defer mc.statsMutex.RUnlock()

	// 计算当前键数量
	var keyCount int64
	mc.data.Range(func(_, _ interface{}) bool {
		keyCount++
		return true
	})

	stats := mc.stats
	stats.KeyCount = keyCount

	return stats, nil
}

// 更新统计信息的辅助方法
func (mc *memoryCache) incrementHitCount() {
	mc.statsMutex.Lock()
	defer mc.statsMutex.Unlock()
	mc.stats.HitCount++
}

func (mc *memoryCache) incrementMissCount() {
	mc.statsMutex.Lock()
	defer mc.statsMutex.Unlock()
	mc.stats.MissCount++
}

func (mc *memoryCache) incrementExpiredCount() {
	mc.statsMutex.Lock()
	defer mc.statsMutex.Unlock()
	mc.stats.ExpiredCount++
}

func (mc *memoryCache) incrementEvictedCount() {
	mc.statsMutex.Lock()
	defer mc.statsMutex.Unlock()
	mc.stats.EvictedCount++
}

// wildcardMatch 实现简单的通配符匹配
// pattern中可以使用*匹配任意字符，?匹配单个字符
func wildcardMatch(s, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// 这里实现一个简单的通配符匹配
	// 在实际场景中可能需要更复杂的实现
	// 或者使用现有的库

	return false // 简化版本，仅支持"*"完全匹配
}
