package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sunliang711/ez-go/ezcache"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	CreateAt int64  `json:"create_at"`
}

func main() {
	fmt.Println("ezcache示例程序")
	fmt.Println("1 - 内存缓存演示")
	fmt.Println("2 - Redis缓存演示 (需要Redis服务)")
	fmt.Print("请选择要运行的示例 (默认1): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "2":
		demoRedisCache()
	default:
		demoMemoryCache()
	}
}

func demoMemoryCache() {
	fmt.Println("开始内存缓存演示...")

	// 创建内存缓存实例
	cache, err := ezcache.NewCache(
		ezcache.WithCacheType(ezcache.MemoryCache),
		ezcache.WithDefaultTTL(time.Minute*5),
		ezcache.WithCleanupInterval(time.Second*30),
		ezcache.WithEnableLog(true),
	)
	if err != nil {
		fmt.Printf("创建内存缓存失败: %v\n", err)
		return
	}

	ctx := context.Background()

	// 基本操作
	demoBasicOperations(ctx, cache)

	// 批量操作
	demoBatchOperations(ctx, cache)

	// 过期时间操作
	demoTTLOperations(ctx, cache)

	// 统计信息
	demoStats(ctx, cache)

	fmt.Println("内存缓存演示结束")
}

func demoRedisCache() {
	fmt.Println("开始Redis缓存演示...")

	// 获取Redis连接信息
	fmt.Println("请输入Redis连接信息 (不填则使用默认值)")

	fmt.Print("Redis地址 (默认localhost:6379): ")
	redisAddr := readInput("localhost:6379")

	fmt.Print("Redis密码 (默认无): ")
	redisPassword := readInput("")

	fmt.Print("Redis数据库 (默认0): ")
	redisDB := 0
	fmt.Scanln(&redisDB)

	// 创建Redis缓存实例
	cache, err := ezcache.NewCache(
		ezcache.WithCacheType(ezcache.RedisCache),
		ezcache.WithRedisOptions(redisAddr, "", redisPassword, redisDB),
		ezcache.WithRedisKeyPrefix("ezcache"),
		ezcache.WithDefaultTTL(time.Minute*5),
		ezcache.WithEnableLog(true),
	)
	if err != nil {
		fmt.Printf("创建Redis缓存失败: %v\n", err)
		return
	}

	fmt.Println("Redis连接成功!")

	ctx := context.Background()

	// 基本操作
	demoBasicOperations(ctx, cache)

	// 批量操作
	demoBatchOperations(ctx, cache)

	// 过期时间操作
	demoTTLOperations(ctx, cache)

	// 统计信息
	demoStats(ctx, cache)

	// 询问是否清空缓存
	fmt.Print("是否清空缓存? (y/N): ")
	var cleanChoice string
	fmt.Scanln(&cleanChoice)

	if cleanChoice == "y" || cleanChoice == "Y" {
		err = cache.Clear(ctx)
		if err != nil {
			fmt.Printf("清空Redis缓存失败: %v\n", err)
		} else {
			fmt.Println("Redis缓存已清空")
		}
	}

	fmt.Println("Redis缓存演示结束")
}

// readInput 读取用户输入，如果为空则返回默认值
func readInput(defaultValue string) string {
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return defaultValue
	}
	return input
}

func demoBasicOperations(ctx context.Context, cache ezcache.Cache) {
	// 设置一个值
	user := User{
		ID:       1,
		Name:     "张三",
		Email:    "zhangsan@example.com",
		CreateAt: time.Now().Unix(),
	}
	err := cache.Set(ctx, "user:1", user, time.Minute)
	if err != nil {
		fmt.Printf("设置缓存失败: %v\n", err)
		return
	}
	fmt.Println("设置用户缓存成功")

	// 获取值
	var retrievedUser User
	err = cache.Get(ctx, "user:1", &retrievedUser)
	if err != nil {
		fmt.Printf("获取缓存失败: %v\n", err)
		return
	}
	fmt.Printf("获取用户成功: %s (ID: %d)\n", retrievedUser.Name, retrievedUser.ID)

	// 检查键是否存在
	exists, err := cache.Exists(ctx, "user:1")
	if err != nil {
		fmt.Printf("检查键存在失败: %v\n", err)
		return
	}
	fmt.Printf("键 'user:1' 是否存在: %v\n", exists)

	// 检查一个不存在的键
	exists, err = cache.Exists(ctx, "user:999")
	if err != nil {
		fmt.Printf("检查键存在失败: %v\n", err)
		return
	}
	fmt.Printf("键 'user:999' 是否存在: %v\n", exists)

	// 删除键
	err = cache.Delete(ctx, "user:1")
	if err != nil {
		fmt.Printf("删除键失败: %v\n", err)
		return
	}
	fmt.Println("删除键 'user:1' 成功")

	// 验证已删除
	err = cache.Get(ctx, "user:1", &retrievedUser)
	if err != nil {
		fmt.Printf("预期的错误（键已删除）: %v\n", err)
	}
}

func demoBatchOperations(ctx context.Context, cache ezcache.Cache) {
	// 批量设置
	users := map[string]interface{}{
		"user:2": User{ID: 2, Name: "李四", Email: "lisi@example.com", CreateAt: time.Now().Unix()},
		"user:3": User{ID: 3, Name: "王五", Email: "wangwu@example.com", CreateAt: time.Now().Unix()},
		"user:4": User{ID: 4, Name: "赵六", Email: "zhaoliu@example.com", CreateAt: time.Now().Unix()},
	}
	err := cache.MSet(ctx, users, time.Minute)
	if err != nil {
		fmt.Printf("批量设置缓存失败: %v\n", err)
		return
	}
	fmt.Println("批量设置用户缓存成功")

	// 批量获取
	keys := []string{"user:2", "user:3", "user:4", "user:999"}
	result, err := cache.MGet(ctx, keys)
	if err != nil {
		fmt.Printf("批量获取缓存失败: %v\n", err)
		return
	}

	fmt.Printf("批量获取结果，共 %d 个键\n", len(result))
	for key, value := range result {
		fmt.Printf("键: %s, 值: %+v\n", key, value)
	}

	// 获取所有键
	allKeys, err := cache.Keys(ctx, "user:*")
	if err != nil {
		fmt.Printf("获取所有键失败: %v\n", err)
		return
	}
	fmt.Printf("所有键: %v\n", allKeys)
}

func demoTTLOperations(ctx context.Context, cache ezcache.Cache) {
	// 设置一个短时间过期的键
	err := cache.Set(ctx, "short-lived", "我将很快过期", time.Second*3)
	if err != nil {
		fmt.Printf("设置短期缓存失败: %v\n", err)
		return
	}
	fmt.Println("设置短期缓存成功")

	// 获取TTL
	ttl, err := cache.TTL(ctx, "short-lived")
	if err != nil {
		fmt.Printf("获取TTL失败: %v\n", err)
		return
	}
	fmt.Printf("键 'short-lived' 的TTL: %v\n", ttl)

	// 修改过期时间
	err = cache.Expire(ctx, "short-lived", time.Second*10)
	if err != nil {
		fmt.Printf("修改过期时间失败: %v\n", err)
		return
	}
	fmt.Println("修改过期时间成功")

	// 再次获取TTL
	ttl, err = cache.TTL(ctx, "short-lived")
	if err != nil {
		fmt.Printf("获取TTL失败: %v\n", err)
		return
	}
	fmt.Printf("修改后键 'short-lived' 的TTL: %v\n", ttl)

	// 等待键过期
	fmt.Println("等待键过期...")
	time.Sleep(time.Second * 11)

	// 检查键是否已过期
	var value string
	err = cache.Get(ctx, "short-lived", &value)
	if err != nil {
		fmt.Printf("预期的错误（键已过期）: %v\n", err)
	} else {
		fmt.Printf("键仍然存在，值: %s\n", value)
	}
}

func demoStats(ctx context.Context, cache ezcache.Cache) {
	// 获取统计信息
	stats, err := cache.Stats(ctx)
	if err != nil {
		fmt.Printf("获取统计信息失败: %v\n", err)
		return
	}

	fmt.Println("缓存统计信息:")
	fmt.Printf("- 键数量: %d\n", stats.KeyCount)
	fmt.Printf("- 命中次数: %d\n", stats.HitCount)
	fmt.Printf("- 未命中次数: %d\n", stats.MissCount)
	fmt.Printf("- 淘汰次数: %d\n", stats.EvictedCount)
	fmt.Printf("- 过期次数: %d\n", stats.ExpiredCount)

	if len(stats.Details) > 0 {
		fmt.Printf("- 详细信息: %+v\n", stats.Details)
	}
}
