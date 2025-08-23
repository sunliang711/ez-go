# EzRocketMQ

EzRocketMQ 是一个基于 Apache RocketMQ Go 客户端的简化封装库，提供了更简单易用的 API 来操作 RocketMQ。

## 特性

- 简化的 API 设计，降低使用门槛
- 支持同步、异步和单向消息发送
- 支持集群和广播消费模式
- 支持延时消息
- 支持消息重试机制
- 内置日志功能
- 优雅的启动和关闭
- 支持认证机制
- 支持创建时传递认证信息
- 支持默认配置，简化生产者和消费者配置
- 可自定义默认配置值

## 安装

```bash
go get github.com/sunliang711/ez-go/ezrocketmq
```

## 快速开始

### 创建 RocketMQ 实例

```go
import (
    "github.com/sunliang711/ez-go/ezrocketmq"
    "github.com/apache/rocketmq-client-go/v2/primitive"
)

// RocketMQ NameServer 地址列表
nameServers := []string{"127.0.0.1:9876"}

// 方式1: 创建无认证的 RocketMQ 实例
rocketMQ, err := ezrocketmq.NewRocketMQ(nameServers, "my-instance", "", true)
if err != nil {
    log.Fatal(err)
}

// 方式2: 创建时直接传递认证信息
credentials := &primitive.Credentials{
    AccessKey:     "your-access-key",
    SecretKey:     "your-secret-key",
    SecurityToken: "your-security-token", // 可选
}
rocketMQ, err := ezrocketmq.NewRocketMQ(nameServers, "my-instance", "", true, credentials)
if err != nil {
    log.Fatal(err)
}

// 方式3: 创建后设置认证信息
rocketMQ.SetCredentials("accessKey", "secretKey", "")
```

### 生产者示例

```go
// 方式1: 使用默认配置添加生产者（推荐）
err = rocketMQ.AddProducer("test-topic", "producer-group", nil)
if err != nil {
    log.Fatal(err)
}

// 方式2: 使用自定义配置添加生产者
producerConfig := &ezrocketmq.ProducerConfig{
    MaxMessageSize: 4 * 1024 * 1024, // 4MB
    SendMsgTimeout: 3000,            // 3秒
    RetryTimes:     3,               // 重试3次
    Tags:           []string{"tag1", "tag2"},
}
err = rocketMQ.AddProducer("custom-topic", "custom-producer-group", producerConfig)
if err != nil {
    log.Fatal(err)
}

// 启动
err = rocketMQ.Start()
if err != nil {
    log.Fatal(err)
}

// 发送消息
sendOpts := &ezrocketmq.SendOptions{
    Tag:  "tag1",
    Keys: []string{"key1"},
    Properties: map[string]string{
        "custom": "value",
    },
}

result, err := rocketMQ.Send("test-topic", []byte("Hello RocketMQ"), sendOpts)
if err != nil {
    log.Printf("Send failed: %v", err)
} else {
    log.Printf("Message sent: %s", result.MessageID)
}
```

### 消费者示例

```go
import (
    "context"
    "github.com/apache/rocketmq-client-go/v2/primitive"
    "github.com/sunliang711/ez-go/ezrocketmq"
)

// 定义消息处理函数
messageHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
    for _, msg := range msgs {
        log.Printf("Received: %s", string(msg.Body))
        // 处理消息逻辑
    }
    return ezrocketmq.ConsumeSuccess, nil
}

// 方式1: 使用默认配置添加消费者（推荐）
err = rocketMQ.AddConsumer("test-topic", "consumer-group", messageHandler, nil)
if err != nil {
    log.Fatal(err)
}

// 方式2: 使用自定义配置添加消费者
consumerConfig := &ezrocketmq.ConsumerConfig{
    Tags:                []string{"tag1"}, // 订阅的标签，空数组表示订阅所有
    ConsumeFromWhere:    ezrocketmq.ConsumeFromLastOffset,
    ConsumeMode:         ezrocketmq.Clustering,
    MaxReconsumeTimes:   16,
    ConsumeTimeout:      15,
    PullInterval:        1000,
    PullBatchSize:       32,
    MaxCachedMessageNum: 1000,
}
err = rocketMQ.AddConsumer("custom-topic", "custom-consumer-group", messageHandler, consumerConfig)
if err != nil {
    log.Fatal(err)
}

// 启动
err = rocketMQ.Start()
if err != nil {
    log.Fatal(err)
}

// 程序退出时停止
defer rocketMQ.Stop()
```

## 默认配置管理

### 设置默认生产者配置

```go
// 设置自定义的默认生产者配置
customProducerDefaults := ezrocketmq.ProducerConfig{
    MaxMessageSize: 8 * 1024 * 1024, // 8MB
    SendMsgTimeout: 5000,            // 5秒
    RetryTimes:     5,
    Tags:           []string{"default", "producer"},
    Properties:     map[string]string{"env": "production"},
}
ezrocketmq.SetDefaultProducerConfig(customProducerDefaults)

// 获取当前的默认生产者配置
currentDefaults := ezrocketmq.GetDefaultProducerConfig()
```

### 设置默认消费者配置

```go
// 设置自定义的默认消费者配置
customConsumerDefaults := ezrocketmq.ConsumerConfig{
    ConsumeFromWhere:    ezrocketmq.ConsumeFromLastOffset,
    ConsumeMode:         ezrocketmq.Clustering,
    MaxReconsumeTimes:   20,
    ConsumeTimeout:      30, // 30分钟
    PullInterval:        2000,
    PullBatchSize:       64,
    MaxCachedMessageNum: 2000,
    Tags:                []string{"default", "consumer"},
    Properties:          map[string]string{"env": "production"},
}
ezrocketmq.SetDefaultConsumerConfig(customConsumerDefaults)

// 获取当前的默认消费者配置
currentDefaults := ezrocketmq.GetDefaultConsumerConfig()
```

### 使用默认配置的优势

1. **简化代码**：只需传递必要参数（topic、groupName、handler）
2. **统一配置**：所有生产者/消费者使用相同的默认配置
3. **易于维护**：修改默认配置即可影响所有使用默认配置的实例
4. **渐进式配置**：可以只覆盖需要修改的配置项

## 消息发送方式

### 同步发送

```go
result, err := rocketMQ.Send("topic", []byte("message"), &ezrocketmq.SendOptions{
    Tag: "tag1",
    Keys: []string{"key1"},
})
```

### 异步发送

```go
err := rocketMQ.SendAsync("topic", []byte("message"), nil, func(result *ezrocketmq.SendResult, err error) {
    if err != nil {
        log.Printf("Send failed: %v", err)
    } else {
        log.Printf("Send success: %s", result.MessageID)
    }
})
```

### 单向发送

```go
err := rocketMQ.SendOneWay("topic", []byte("message"), &ezrocketmq.SendOptions{
    Tag: "tag1",
})
```

### 延时消息

```go
result, err := rocketMQ.Send("topic", []byte("delay message"), &ezrocketmq.SendOptions{
    Tag:        "delay",
    DelayLevel: 3, // 延时级别3，对应10秒延时
})
```

## 消费模式

### 集群消费模式（默认）

```go
consumerConfig := ezrocketmq.ConsumerConfig{
    ConsumeMode: ezrocketmq.Clustering,
    // 其他配置...
}
```

### 广播消费模式

```go
consumerConfig := ezrocketmq.ConsumerConfig{
    ConsumeMode: ezrocketmq.Broadcasting,
    // 其他配置...
}
```

## 消费起始位置

```go
// 从队列头开始消费
ConsumeFromWhere: ezrocketmq.ConsumeFromFirstOffset

// 从队列尾开始消费（默认）
ConsumeFromWhere: ezrocketmq.ConsumeFromLastOffset

// 从存储的消费位点开始消费
ConsumeFromWhere: ezrocketmq.ConsumeFromStoredOffset
```

## 错误处理

### 消费重试

```go
messageHandler := func(ctx context.Context, msgs []*primitive.MessageExt) (ezrocketmq.ConsumeResult, error) {
    for _, msg := range msgs {
        err := processMessage(msg)
        if err != nil {
            log.Printf("Process failed: %v", err)
            return ezrocketmq.ConsumeRetryLater, nil // 重试
        }
    }
    return ezrocketmq.ConsumeSuccess, nil
}
```

### 自定义错误处理

```go
result, err := rocketMQ.Send("topic", []byte("message"), nil)
if err != nil {
    if rocketErr, ok := err.(*ezrocketmq.RocketMQError); ok {
        log.Printf("RocketMQ Error - Code: %d, Message: %s", rocketErr.Code, rocketErr.Message)
    }
}
```

## 状态监控

```go
// 检查生产者状态
state := rocketMQ.GetProducerState("topic")
switch state {
case ezrocketmq.ProducerStateStarted:
    log.Println("Producer is running")
case ezrocketmq.ProducerStateCreated:
    log.Println("Producer is created but not started")
case ezrocketmq.ProducerStateStopped:
    log.Println("Producer is stopped")
}

// 检查消费者状态
state := rocketMQ.GetConsumerState("topic")

// 检查客户端是否启动
if rocketMQ.IsStarted() {
    log.Println("RocketMQ client is running")
}
```

## 配置选项

### ProducerConfig

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| MaxMessageSize | int | 4MB | 最大消息大小 |
| SendMsgTimeout | int | 3000 | 发送超时时间(毫秒) |
| RetryTimes | int | 3 | 重试次数 |
| Tags | []string | nil | 支持的标签列表 |
| Properties | map[string]string | nil | 自定义属性 |

### ConsumerConfig

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Tags | []string | nil | 订阅的标签，空表示订阅所有 |
| ConsumeFromWhere | ConsumeFromWhere | ConsumeFromLastOffset | 消费起始位置 |
| ConsumeMode | ConsumeMode | Clustering | 消费模式 |
| MaxReconsumeTimes | int32 | 16 | 最大重消费次数 |
| ConsumeTimeout | int64 | 15 | 消费超时时间(分钟) |
| PullInterval | int64 | 1000 | 拉取间隔(毫秒) |
| PullBatchSize | int32 | 32 | 批量拉取大小 |
| MaxCachedMessageNum | int32 | 1000 | 最大缓存消息数量 |
| Properties | map[string]string | nil | 自定义属性 |
| Interceptors | []primitive.Interceptor | nil | 拦截器 |

### SendOptions

| 字段 | 类型 | 说明 |
|------|------|------|
| Tag | string | 消息标签 |
| Keys | []string | 消息关键字 |
| Properties | map[string]string | 自定义属性 |
| DelayLevel | int | 延时级别(1-18) |
| Timeout | int64 | 发送超时时间(毫秒) |

## 延时级别对照表

| 延时级别 | 延时时间 |
|----------|----------|
| 1 | 1s |
| 2 | 5s |
| 3 | 10s |
| 4 | 30s |
| 5 | 1m |
| 6 | 2m |
| 7 | 3m |
| 8 | 4m |
| 9 | 5m |
| 10 | 6m |
| 11 | 7m |
| 12 | 8m |
| 13 | 9m |
| 14 | 10m |
| 15 | 20m |
| 16 | 30m |
| 17 | 1h |
| 18 | 2h |

## 最佳实践

1. **使用默认配置**：优先使用默认配置，减少代码复杂度
2. **设置全局默认配置**：在应用启动时设置适合的默认配置
3. **合理设置消费者数量**：根据消息量和处理能力配置适当的消费者数量
4. **错误处理**：实现完善的错误处理和重试机制
5. **消息幂等**：确保消息处理的幂等性
6. **批量处理**：合理设置批量大小以提高性能
7. **监控**：监控消息积压、消费延迟等指标
8. **优雅关闭**：确保程序退出时调用 `Stop()` 方法
9. **认证安全**：在生产环境中使用认证机制，保护消息安全

## 示例代码

完整的示例代码请参考：
- [生产者示例](examples/producer/main.go)
- [消费者示例](examples/consumer/main.go)
- [新功能示例](examples/new_features_example.go)

## 依赖

- [Apache RocketMQ Go Client](https://github.com/apache/rocketmq-client-go)
- [Zerolog](https://github.com/rs/zerolog)

## 许可证

MIT License