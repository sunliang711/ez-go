package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/sunliang711/ez-go/ezrocketmq"
)

func main() {
	// Example 1: Create RocketMQ with credentials
	credentials := &primitive.Credentials{
		AccessKey:     "your-access-key",
		SecretKey:     "your-secret-key",
		SecurityToken: "your-security-token", // optional
	}

	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := ezrocketmq.NewRocketMQ(nameServers, "example-instance", "", true, credentials)
	if err != nil {
		log.Fatalf("Failed to create RocketMQ: %v", err)
	}

	// Example 2: Set custom default configurations
	customProducerDefaults := ezrocketmq.ProducerConfig{
		MaxMessageSize: 8 * 1024 * 1024, // 8MB
		SendMsgTimeout: 5000,            // 5 seconds
		RetryTimes:     5,
		Tags:           []string{"default", "producer"},
		Properties:     map[string]string{"env": "production"},
	}
	ezrocketmq.SetDefaultProducerConfig(customProducerDefaults)

	customConsumerDefaults := ezrocketmq.ConsumerConfig{
		ConsumeFromWhere:    ezrocketmq.ConsumeFromLastOffset,
		ConsumeMode:         ezrocketmq.Clustering,
		MaxReconsumeTimes:   20,
		ConsumeTimeout:      30, // 30 minutes
		PullInterval:        2000,
		PullBatchSize:       64,
		MaxCachedMessageNum: 2000,
		Tags:                []string{"default", "consumer"},
		Properties:          map[string]string{"env": "production"},
	}
	ezrocketmq.SetDefaultConsumerConfig(customConsumerDefaults)

	// Example 3: Add producer with default config (only topic and groupName required)
	err = rocketMQ.AddProducer("test-topic", "producer-group", nil)
	if err != nil {
		log.Fatalf("Failed to add producer with defaults: %v", err)
	}

	// Example 4: Add producer with custom config (overrides defaults)
	customProducerConfig := &ezrocketmq.ProducerConfig{
		MaxMessageSize: 2 * 1024 * 1024, // 2MB, overrides default 8MB
		SendMsgTimeout: 3000,            // 3 seconds, overrides default 5 seconds
		// RetryTimes not set, will use default value of 5
		Tags: []string{"custom", "important"}, // overrides default tags
	}
	err = rocketMQ.AddProducer("important-topic", "important-producer-group", customProducerConfig)
	if err != nil {
		log.Fatalf("Failed to add producer with custom config: %v", err)
	}

	// Example 5: Add consumer with default config (only topic, groupName, and handler required)
	defaultHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
		for _, msg := range msgs {
			fmt.Printf("Received message with default config: %s\n", string(msg.Body))
		}
		return ezrocketmq.ConsumeSuccess, nil
	}

	err = rocketMQ.AddConsumer("test-topic", "consumer-group", defaultHandler, nil)
	if err != nil {
		log.Fatalf("Failed to add consumer with defaults: %v", err)
	}

	// Example 6: Add consumer with custom config (overrides defaults)
	customHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
		for _, msg := range msgs {
			fmt.Printf("Received important message: %s\n", string(msg.Body))
		}
		return ezrocketmq.ConsumeSuccess, nil
	}

	customConsumerConfig := &ezrocketmq.ConsumerConfig{
		ConsumeFromWhere:  ezrocketmq.ConsumeFromFirstOffset, // override default
		ConsumeMode:       ezrocketmq.Broadcasting,           // override default
		MaxReconsumeTimes: 10,                                // override default
		// Other fields will use default values
		Tags: []string{"important", "urgent"}, // override default tags
	}

	err = rocketMQ.AddConsumer("important-topic", "important-consumer-group", customHandler, customConsumerConfig)
	if err != nil {
		log.Fatalf("Failed to add consumer with custom config: %v", err)
	}

	// Example 7: Start RocketMQ
	err = rocketMQ.Start()
	if err != nil {
		log.Fatalf("Failed to start RocketMQ: %v", err)
	}

	// Example 8: Send messages
	// Send to topic with default producer config
	result, err := rocketMQ.Send("test-topic", []byte("Hello from default producer"), &ezrocketmq.SendOptions{
		Tag:  "greeting",
		Keys: []string{"msg1"},
	})
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	} else {
		fmt.Printf("Message sent successfully: %s\n", result.MessageID)
	}

	// Send to topic with custom producer config
	result, err = rocketMQ.Send("important-topic", []byte("Important message"), &ezrocketmq.SendOptions{
		Tag:  "important",
		Keys: []string{"msg2"},
	})
	if err != nil {
		log.Printf("Failed to send important message: %v", err)
	} else {
		fmt.Printf("Important message sent successfully: %s\n", result.MessageID)
	}

	// Example 9: Get current default configurations
	currentProducerDefaults := ezrocketmq.GetDefaultProducerConfig()
	fmt.Printf("Current producer defaults - MaxMessageSize: %d, SendMsgTimeout: %d\n",
		currentProducerDefaults.MaxMessageSize, currentProducerDefaults.SendMsgTimeout)

	currentConsumerDefaults := ezrocketmq.GetDefaultConsumerConfig()
	fmt.Printf("Current consumer defaults - MaxReconsumeTimes: %d, PullBatchSize: %d\n",
		currentConsumerDefaults.MaxReconsumeTimes, currentConsumerDefaults.PullBatchSize)

	// Wait for messages
	time.Sleep(30 * time.Second)

	// Stop RocketMQ
	rocketMQ.Stop()
	fmt.Println("RocketMQ stopped")
}
