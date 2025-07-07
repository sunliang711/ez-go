package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sunliang711/ez-go/ezlock"
)

func main() {
	fmt.Println("=== Redis分布式锁示例 ===")

	// 获取Redis连接信息
	fmt.Println("请输入Redis连接信息 (不填则使用默认值)")

	fmt.Print("Redis地址 (默认localhost:6379): ")
	redisAddr := readInput("localhost:6379")

	fmt.Print("Redis密码 (默认无): ")
	redisPassword := readInput("")

	fmt.Print("Redis数据库 (默认0): ")
	redisDB := 0
	fmt.Scanln(&redisDB)

	// 创建锁实例
	lock, err := ezlock.NewRedisLock(
		ezlock.WithRedisOptions(redisAddr, redisPassword, redisDB),
		ezlock.WithRedisKeyPrefix("ezlock"),
		ezlock.WithRetryOptions(3, time.Millisecond*100),
		ezlock.WithEnableLog(true),
	)
	if err != nil {
		fmt.Printf("创建锁失败: %v\n", err)
		return
	}

	fmt.Println("Redis连接成功!")

	// 创建上下文，用于优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听中断信号
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		<-signalCh
		fmt.Println("\n收到中断信号，正在退出...")
		cancel()
	}()

	// 提供多种示例
	fmt.Println("\n请选择要运行的示例:")
	fmt.Println("1. 基本分布式锁示例")
	fmt.Println("2. 多客户端竞争锁示例")
	fmt.Println("3. 锁续约示例")
	fmt.Print("请选择 (1-3): ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		basicLockDemo(ctx, lock)
	case 2:
		multiClientDemo(ctx, lock)
	case 3:
		lockRenewalDemo(ctx, lock)
	default:
		fmt.Println("无效的选择")
	}
}

// 基本分布式锁示例
func basicLockDemo(ctx context.Context, lock ezlock.Lock) {
	fmt.Println("\n=== 运行基本分布式锁示例 ===")

	// 生成唯一的客户端ID
	clientID := uuid.New().String()
	fmt.Printf("客户端ID: %s\n", clientID)

	// 资源键
	key := "resource:basic-demo"
	// 锁过期时间
	ttl := time.Second * 10

	fmt.Println("尝试获取锁...")
	err := lock.Acquire(ctx, key, clientID, ttl)
	if err != nil {
		fmt.Printf("获取锁失败: %v\n", err)
		return
	}

	fmt.Println("成功获取锁!")
	fmt.Println("锁将在10秒后自动过期")
	fmt.Println("按Ctrl+C退出程序")

	// 每秒检查一次锁状态
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			// 释放锁并退出
			fmt.Println("正在释放锁...")
			if err := lock.Release(ctx, key, clientID); err != nil {
				fmt.Printf("释放锁失败: %v\n", err)
			} else {
				fmt.Println("锁已释放")
			}
			return
		case <-ticker.C:
			counter++
			fmt.Printf("锁已持有 %d 秒...\n", counter)

			// 检查锁状态
			locked, owner, err := lock.IsLocked(ctx, key)
			if err != nil {
				fmt.Printf("检查锁状态失败: %v\n", err)
				continue
			}

			if !locked {
				fmt.Println("锁已丢失!")
				return
			} else if owner != clientID {
				fmt.Printf("锁被其他客户端获取: %s\n", owner)
				return
			}
		}
	}
}

// 多客户端竞争锁示例
func multiClientDemo(ctx context.Context, lock ezlock.Lock) {
	fmt.Println("\n=== 运行多客户端竞争锁示例 ===")

	// 创建5个模拟客户端
	numClients := 5
	var wg sync.WaitGroup
	wg.Add(numClients)

	// 共享的资源键
	key := "resource:multi-client-demo"

	// 启动多个客户端并发获取锁
	for i := 1; i <= numClients; i++ {
		clientID := fmt.Sprintf("client-%d", i)

		go func(id string) {
			defer wg.Done()

			// 随机延迟，模拟不同时间的请求
			time.Sleep(time.Duration(100+i*50) * time.Millisecond)

			fmt.Printf("[%s] 尝试获取锁...\n", id)
			err := lock.Acquire(ctx, key, id, time.Second*5)
			if err != nil {
				fmt.Printf("[%s] 获取锁失败: %v\n", id, err)
				return
			}

			fmt.Printf("[%s] 成功获取锁! 持有3秒...\n", id)
			time.Sleep(time.Second * 3)

			// 释放锁
			if err := lock.Release(ctx, key, id); err != nil {
				fmt.Printf("[%s] 释放锁失败: %v\n", id, err)
			} else {
				fmt.Printf("[%s] 锁已释放\n", id)
			}
		}(clientID)
	}

	// 等待所有客户端完成
	wg.Wait()
	fmt.Println("所有客户端已完成")
}

// 锁续约示例
func lockRenewalDemo(ctx context.Context, lock ezlock.Lock) {
	fmt.Println("\n=== 运行锁续约示例 ===")

	// 生成唯一的客户端ID
	clientID := uuid.New().String()
	fmt.Printf("客户端ID: %s\n", clientID)

	// 资源键
	key := "resource:renewal-demo"
	// 初始锁过期时间 - 设置较短以便演示续约
	ttl := time.Second * 3

	fmt.Println("尝试获取锁...")
	err := lock.Acquire(ctx, key, clientID, ttl)
	if err != nil {
		fmt.Printf("获取锁失败: %v\n", err)
		return
	}

	fmt.Println("成功获取锁!")
	fmt.Println("锁将每2秒自动续约一次")
	fmt.Println("按Ctrl+C退出程序")

	// 每秒检查一次锁状态
	stateTicker := time.NewTicker(time.Second)
	defer stateTicker.Stop()

	// 每2秒续约一次锁
	renewalTicker := time.NewTicker(time.Second * 2)
	defer renewalTicker.Stop()

	counter := 0
	for {
		select {
		case <-ctx.Done():
			// 释放锁并退出
			fmt.Println("正在释放锁...")
			if err := lock.Release(ctx, key, clientID); err != nil {
				fmt.Printf("释放锁失败: %v\n", err)
			} else {
				fmt.Println("锁已释放")
			}
			return
		case <-stateTicker.C:
			counter++
			fmt.Printf("锁已持有 %d 秒...\n", counter)

			// 检查锁状态
			locked, _, _ := lock.IsLocked(ctx, key)
			if !locked {
				fmt.Println("锁已丢失!")
				return
			}
		case <-renewalTicker.C:
			// 续约锁
			fmt.Println("正在续约锁...")
			err := lock.Refresh(ctx, key, clientID, ttl)
			if err != nil {
				fmt.Printf("续约失败: %v\n", err)
				return
			}
			fmt.Printf("续约成功, 新的过期时间为 %v\n", ttl)
		}
	}
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
