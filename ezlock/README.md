# EZLock - 简单易用的分布式锁库

EZLock是一个基于Redis的分布式锁库，提供了简单易用的API来实现分布式锁功能。

## 特性

- 支持基于Redis的分布式锁
- 防止锁被非持有者释放
- 支持锁续约
- 支持获取锁时自动重试
- 简洁明了的API

## 安装

```bash
go get github.com/yourusername/ez-go/ezlock
```

## 使用方法

### 基本用法

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/ez-go/ezlock"
)

func main() {
	// 创建锁实例
	lock, err := ezlock.NewRedisLock(
		ezlock.WithRedisOptions("localhost:6379", "", 0),
		ezlock.WithRedisKeyPrefix("myapp"),
		ezlock.WithRetryOptions(3, time.Millisecond*100),
		ezlock.WithEnableLog(true),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	key := "my-resource"
	value := "client-1" // 客户端的唯一标识
	ttl := time.Second * 10

	// 获取锁
	err = lock.Acquire(ctx, key, value, ttl)
	if err != nil {
		if err == ezlock.ErrLockNotAcquired {
			fmt.Println("无法获取锁，资源正在被其他客户端使用")
		} else {
			fmt.Printf("获取锁失败: %v\n", err)
		}
		return
	}

	fmt.Println("成功获取锁，正在处理任务...")

	// 业务逻辑处理
	// ...

	// 刷新锁（延长锁的过期时间）
	err = lock.Refresh(ctx, key, value, ttl)
	if err != nil {
		fmt.Printf("刷新锁失败: %v\n", err)
	}

	// 业务逻辑处理
	// ...

	// 释放锁
	err = lock.Release(ctx, key, value)
	if err != nil {
		fmt.Printf("释放锁失败: %v\n", err)
	}

	fmt.Println("锁已释放")
}
```

### 检查锁状态

```go
// 检查锁是否被占用
locked, owner, err := lock.IsLocked(ctx, key)
if err != nil {
	fmt.Printf("检查锁状态失败: %v\n", err)
	return
}

if locked {
	fmt.Printf("资源正在被客户端 %s 使用\n", owner)
} else {
	fmt.Println("资源当前可用")
}
```

## 注意事项

1. Redis分布式锁不提供完全的强一致性保证，在某些极端情况下可能会失效（如时钟漂移）。
2. 合理设置锁的过期时间，避免因客户端崩溃导致锁无法释放。
3. 对于关键业务，建议使用更强大的分布式协调服务如etcd或ZooKeeper。

## 依赖

- [redis/go-redis/v9](https://github.com/redis/go-redis)

## 许可证

MIT 