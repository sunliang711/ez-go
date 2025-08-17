package main

import (
	"fmt"
	"log"
	"time"

	ezrocketmq "github.com/sunliang711/ez-go/ezrocketmq"
)

func main() {
	// RocketMQ NameServer地址列表
	nameServers := []string{"10.2.9.64:9876"}

	// 创建RocketMQ实例
	rocketMQ, err := ezrocketmq.NewRocketMQ(nameServers, "producer-instance-1", "", true)
	if err != nil {
		log.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	// 如果需要认证，可以设置凭据
	// rocketMQ.SetCredentials("accessKey", "secretKey", "")

	// 定义Topic和生产者组
	topic := "test-topic"
	groupName := "test-producer-group"

	// 配置生产者
	producerConfig := ezrocketmq.ProducerConfig{
		MaxMessageSize: 4 * 1024 * 1024, // 4MB
		SendMsgTimeout: 3000,            // 3秒
		RetryTimes:     3,               // 重试3次
		Tags:           []string{"tag1", "tag2"},
		Properties: map[string]string{
			"producer": "example",
		},
	}

	// 添加生产者
	err = rocketMQ.AddProducer(topic, groupName, producerConfig)
	if err != nil {
		log.Fatalf("Failed to add producer: %v", err)
	}

	// 启动RocketMQ客户端
	err = rocketMQ.Start()
	if err != nil {
		log.Fatalf("Failed to start RocketMQ: %v", err)
	}

	// 确保程序退出时关闭RocketMQ
	defer rocketMQ.Stop()

	log.Println("RocketMQ Producer started successfully")

	// 发送消息示例
	for i := 0; i < 100; i++ {
		message := fmt.Sprintf("Hello RocketMQ! Message %d", i)

		// 发送选项
		sendOpts := &ezrocketmq.SendOptions{
			Tag:  "tag1",
			Keys: []string{fmt.Sprintf("key-%d", i)},
			Properties: map[string]string{
				"messageIndex": fmt.Sprintf("%d", i),
				"timestamp":    fmt.Sprintf("%d", time.Now().Unix()),
			},
			Timeout: 5000, // 5秒超时
		}

		// 同步发送消息
		result, err := rocketMQ.Send(topic, []byte(message), sendOpts)
		if err != nil {
			log.Printf("Failed to send message %d: %v", i, err)
			continue
		}

		log.Printf("Message %d sent successfully - MessageID: %s, QueueID: %d, QueueOffset: %d",
			i, result.MessageID, result.QueueID, result.QueueOffset)

		// 每隔1秒发送一条消息
		time.Sleep(time.Second)
	}

	// 异步发送消息示例
	log.Println("Starting async message sending...")
	for i := 100; i < 110; i++ {
		message := fmt.Sprintf("Async Hello RocketMQ! Message %d", i)

		sendOpts := &ezrocketmq.SendOptions{
			Tag:  "tag2",
			Keys: []string{fmt.Sprintf("async-key-%d", i)},
			Properties: map[string]string{
				"messageIndex": fmt.Sprintf("%d", i),
				"type":         "async",
			},
		}

		// 异步发送消息
		err := rocketMQ.SendAsync(topic, []byte(message), sendOpts, func(result *ezrocketmq.SendResult, err error) {
			if err != nil {
				log.Printf("Async message %d failed: %v", i, err)
			} else {
				log.Printf("Async message %d sent successfully - MessageID: %s", i, result.MessageID)
			}
		})

		if err != nil {
			log.Printf("Failed to send async message %d: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// 单向发送消息示例（不关心发送结果，性能最高）
	log.Println("Starting one-way message sending...")
	for i := 110; i < 120; i++ {
		message := fmt.Sprintf("OneWay Hello RocketMQ! Message %d", i)

		sendOpts := &ezrocketmq.SendOptions{
			Tag:  "tag1",
			Keys: []string{fmt.Sprintf("oneway-key-%d", i)},
			Properties: map[string]string{
				"messageIndex": fmt.Sprintf("%d", i),
				"type":         "oneway",
			},
		}

		// 单向发送消息
		err := rocketMQ.SendOneWay(topic, []byte(message), sendOpts)
		if err != nil {
			log.Printf("Failed to send one-way message %d: %v", i, err)
		} else {
			log.Printf("One-way message %d sent", i)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// 延时消息示例
	log.Println("Sending delay messages...")
	for i := 120; i < 125; i++ {
		message := fmt.Sprintf("Delay Hello RocketMQ! Message %d", i)

		sendOpts := &ezrocketmq.SendOptions{
			Tag:        "delay",
			Keys:       []string{fmt.Sprintf("delay-key-%d", i)},
			DelayLevel: 3, // 延时级别3，对应10秒延时
			Properties: map[string]string{
				"messageIndex": fmt.Sprintf("%d", i),
				"type":         "delay",
			},
		}

		result, err := rocketMQ.Send(topic, []byte(message), sendOpts)
		if err != nil {
			log.Printf("Failed to send delay message %d: %v", i, err)
		} else {
			log.Printf("Delay message %d sent - MessageID: %s (will be consumed after 10 seconds)", i, result.MessageID)
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Println("All messages sent. Waiting for async callbacks...")
	time.Sleep(5 * time.Second)

	log.Println("Producer example completed")
}
