package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	ezrocketmq "github.com/sunliang711/ez-go/ezrocketmq"
)

func main() {
	// RocketMQ NameServer地址列表
	nameServers := []string{"10.2.9.64:9876"}

	// 创建RocketMQ实例
	rocketMQ, err := ezrocketmq.NewRocketMQ(nameServers, "consumer-instance-1", "", true)
	if err != nil {
		log.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	// 如果需要认证，可以设置凭据
	// rocketMQ.SetCredentials("accessKey", "secretKey", "")

	// 定义Topic和消费者组
	topic := "test-topic"
	groupName := "test-consumer-group"

	// 定义消息处理函数
	messageHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
		for _, msg := range msgs {
			log.Printf("Received message - Topic: %s, Tag: %s, Key: %s, Body: %s, MsgID: %s, QueueOffset: %d",
				msg.Topic,
				msg.GetTags(),
				msg.GetKeys(),
				string(msg.Body),
				msg.MsgId,
				msg.QueueOffset,
			)

			// 打印消息属性
			if len(msg.GetProperties()) > 0 {
				log.Printf("Message properties:")
				for k, v := range msg.GetProperties() {
					log.Printf("  %s: %s", k, v)
				}
			}

			// 模拟消息处理时间
			time.Sleep(100 * time.Millisecond)

			// 可以根据消息内容决定消费结果
			// 这里简单地成功消费所有消息
		}

		return ezrocketmq.ConsumeSuccess, nil
	}

	// 配置消费者 - 订阅所有标签
	consumerConfig := ezrocketmq.ConsumerConfig{
		Tags:                []string{}, // 空数组表示订阅所有标签
		ConsumeFromWhere:    ezrocketmq.ConsumeFromLastOffset,
		ConsumeMode:         ezrocketmq.Clustering,
		MaxReconsumeTimes:   16,
		ConsumeTimeout:      15,
		PullInterval:        1000,
		PullBatchSize:       32,
		MaxCachedMessageNum: 1000,
		Properties: map[string]string{
			"consumer": "example",
		},
	}

	// 添加消费者
	err = rocketMQ.AddConsumer(topic, groupName, messageHandler, consumerConfig)
	if err != nil {
		log.Fatalf("Failed to add consumer: %v", err)
	}

	// 再添加一个消费者，只订阅特定标签
	specificTagHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
		for _, msg := range msgs {
			log.Printf("Specific tag consumer - Received message with tag: %s, Body: %s, MsgID: %s",
				msg.GetTags(),
				string(msg.Body),
				msg.MsgId,
			)
		}
		return ezrocketmq.ConsumeSuccess, nil
	}

	specificTagConfig := ezrocketmq.ConsumerConfig{
		Tags:                []string{"delay"}, // 只订阅delay标签的消息
		ConsumeFromWhere:    ezrocketmq.ConsumeFromLastOffset,
		ConsumeMode:         ezrocketmq.Clustering,
		MaxReconsumeTimes:   16,
		ConsumeTimeout:      15,
		PullInterval:        1000,
		PullBatchSize:       10,
		MaxCachedMessageNum: 500,
	}

	// 使用不同的消费者组来避免冲突
	err = rocketMQ.AddConsumer(topic, "delay-consumer-group", specificTagHandler, specificTagConfig)
	if err != nil {
		log.Fatalf("Failed to add specific tag consumer: %v", err)
	}

	// 添加错误处理的消费者示例
	errorHandlerConfig := ezrocketmq.ConsumerConfig{
		Tags:                []string{"error-test"},
		ConsumeFromWhere:    ezrocketmq.ConsumeFromLastOffset,
		ConsumeMode:         ezrocketmq.Clustering,
		MaxReconsumeTimes:   3, // 最多重试3次
		ConsumeTimeout:      5,
		PullInterval:        1000,
		PullBatchSize:       1,
		MaxCachedMessageNum: 100,
	}

	errorHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
		for _, msg := range msgs {
			log.Printf("Error handler - Processing message: %s, Reconsume times: %d",
				string(msg.Body),
				msg.ReconsumeTimes,
			)

			// 模拟处理失败的情况
			if msg.ReconsumeTimes < 2 {
				log.Printf("Simulating processing failure, will retry...")
				return ezrocketmq.ConsumeRetryLater, nil
			}

			log.Printf("Message processed successfully after retries")
		}
		return ezrocketmq.ConsumeSuccess, nil
	}

	err = rocketMQ.AddConsumer(topic, "error-consumer-group", errorHandler, errorHandlerConfig)
	if err != nil {
		log.Fatalf("Failed to add error handler consumer: %v", err)
	}

	// 启动RocketMQ客户端
	err = rocketMQ.Start()
	if err != nil {
		log.Fatalf("Failed to start RocketMQ: %v", err)
	}

	log.Println("RocketMQ Consumer started successfully")
	log.Println("Waiting for messages... Press Ctrl+C to exit")

	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	<-sigChan
	log.Println("Received shutdown signal, stopping consumer...")

	// 停止RocketMQ客户端
	rocketMQ.Stop()
	log.Println("Consumer stopped successfully")
}
