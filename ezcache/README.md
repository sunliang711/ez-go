# ezcache - 高性能轻量级缓存模块

ezcache是ez-go项目的一个模块，提供了简单易用且功能强大的缓存支持。它支持多种缓存后端，统一的API接口，以及丰富的功能。

## 特性

- **多后端支持**：
  - 内存缓存：适用于单机应用
  - Redis缓存：适用于分布式环境
  - 可扩展设计，便于添加更多后端

- **丰富的功能**：
  - 键值存储与检索
  - 批量操作(MGet/MSet)
  - 过期时间管理
  - 通配符键查找
  - 缓存统计

- **性能优化**：
  - 内存缓存使用高效的数据结构
  - 定期清理过期项
  - Redis操作批处理

- **易用性**：
  - 统一的API接口
  - 灵活的配置选项
  - 完整的错误处理

## 快速开始

### 安装

```bash
go get github.com/sunliang711/ez-go/ezcache
```

### 基本使用

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sunliang711/ez-go/ezcache"
)

func main() {
	// 创建内存缓存
	cache, err := ezcache.NewCache(
		ezcache.WithCacheType(ezcache.MemoryCache),
		ezcache.WithDefaultTTL(time.Minute*5),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// 设置缓存
	err = cache.Set(ctx, "hello", "world", time.Minute)
	if err != nil {
		panic(err)
	}

	// 获取缓存
	var value string
	err = cache.Get(ctx, "hello", &value)
	if err != nil {
		panic(err)
	}
	fmt.Println("Value:", value) // Output: Value: world

	// 检查键是否存在
	exists, _ := cache.Exists(ctx, "hello")
	fmt.Println("Exists:", exists) // Output: Exists: true

	// 删除缓存
	cache.Delete(ctx, "hello")
}
```

## 缓存类型

### 内存缓存

适用于单机应用，数据存储在内存中。

```go
cache, err := ezcache.NewCache(
	ezcache.WithCacheType(ezcache.MemoryCache),
	ezcache.WithDefaultTTL(time.Minute*5),
	ezcache.WithCleanupInterval(time.Minute),
	ezcache.WithMaxEntries(10000),
)
```

### Redis缓存

适用于分布式环境，数据存储在Redis中。使用 github.com/redis/go-redis/v9 作为Redis客户端。

```go
cache, err := ezcache.NewCache(
	ezcache.WithCacheType(ezcache.RedisCache),
	ezcache.WithRedisOptions("localhost:6379", "password", 0),
	ezcache.WithRedisKeyPrefix("myapp"),
	ezcache.WithDefaultTTL(time.Minute*5),
)
```

## 高级功能

### 批量操作

```go
// 批量设置
items := map[string]interface{}{
	"key1": "value1",
	"key2": "value2",
	"key3": "value3",
}
err := cache.MSet(ctx, items, time.Minute)

// 批量获取
keys := []string{"key1", "key2", "key3"}
values, err := cache.MGet(ctx, keys)
```

### TTL操作

```go
// 设置过期时间
err := cache.Expire(ctx, "key", time.Minute*10)

// 获取剩余过期时间
ttl, err := cache.TTL(ctx, "key")
```

### 键查找

```go
// 查找所有以"user:"开头的键
keys, err := cache.Keys(ctx, "user:*")
```

### 统计信息

```go
// 获取缓存统计信息
stats, err := cache.Stats(ctx)
fmt.Printf("键数量: %d\n", stats.KeyCount)
fmt.Printf("命中次数: %d\n", stats.HitCount)
fmt.Printf("未命中次数: %d\n", stats.MissCount)
```

## 完整示例

参见 `examples/main.go` 了解更多使用示例。运行示例程序：

```bash
cd examples
go run main.go
```

示例程序支持选择内存缓存或Redis缓存进行演示。

## 注意事项

- 内存缓存不适合存储大量数据，会增加内存压力
- Redis缓存需要正确配置Redis连接信息
- 使用Clear方法时要谨慎，它会清空所有缓存数据

## 待实现功能

- 支持更多缓存后端（如Memcached）
- 缓存穿透保护
- 分级缓存
- 更完善的监控指标 