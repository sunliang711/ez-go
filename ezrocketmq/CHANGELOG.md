# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2024-12-19

### Added
- **Credentials Support in NewRocketMQ**: You can now pass credentials directly when creating a new RocketMQ instance
  ```go
  credentials := &primitive.Credentials{
      AccessKey:     "your-access-key",
      SecretKey:     "your-secret-key",
      SecurityToken: "your-security-token",
  }
  rocketMQ, err := ezrocketmq.NewRocketMQ(nameServers, "instance", "", true, credentials)
  ```

- **Default Configuration System**: Added support for default configurations to simplify usage
  - `SetDefaultProducerConfig()` - Set global default producer configuration
  - `SetDefaultConsumerConfig()` - Set global default consumer configuration
  - `GetDefaultProducerConfig()` - Get current default producer configuration
  - `GetDefaultConsumerConfig()` - Get current default consumer configuration

- **Simplified AddProducer and AddConsumer APIs**: Config parameters are now optional (can be nil)
  - **AddProducer**: Only `topic` and `groupName` are required, config is optional
    ```go
    // Using defaults
    err = rocketMQ.AddProducer("test-topic", "producer-group", nil)
    
    // Using custom config
    config := &ProducerConfig{MaxMessageSize: 8*1024*1024}
    err = rocketMQ.AddProducer("test-topic", "producer-group", config)
    ```
  
  - **AddConsumer**: Only `topic`, `groupName`, and `messageHandler` are required, config is optional
    ```go
    // Using defaults
    err = rocketMQ.AddConsumer("test-topic", "consumer-group", handler, nil)
    
    // Using custom config
    config := &ConsumerConfig{MaxReconsumeTimes: 20}
    err = rocketMQ.AddConsumer("test-topic", "consumer-group", handler, config)
    ```

- **Thread-Safe Default Configuration**: All default configuration operations are thread-safe with proper mutex protection

- **Configuration Inheritance**: When providing partial configurations, unset fields automatically inherit from default values

### Changed
- **BREAKING**: `AddProducer()` method signature changed from `AddProducer(topic, groupName string, config ProducerConfig)` to `AddProducer(topic, groupName string, config *ProducerConfig)`
- **BREAKING**: `AddConsumer()` method signature changed from `AddConsumer(topic, groupName string, handler MessageHandlerFunc, config ConsumerConfig)` to `AddConsumer(topic, groupName string, handler MessageHandlerFunc, config *ConsumerConfig)`
- **BREAKING**: `NewRocketMQ()` method signature changed to accept optional credentials parameter: `NewRocketMQ(nameServers []string, instanceName, namespace string, enableLog bool, credentials ...*primitive.Credentials)`

### Enhanced
- Improved test coverage with new test cases for default configurations
- Enhanced documentation with examples of new features
- Added comprehensive example file demonstrating all new features

### Default Values
#### Producer Defaults
- MaxMessageSize: 4MB
- SendMsgTimeout: 3000ms (3 seconds)
- RetryTimes: 3

#### Consumer Defaults
- ConsumeFromWhere: ConsumeFromLastOffset
- ConsumeMode: Clustering
- MaxReconsumeTimes: 16
- ConsumeTimeout: 15 minutes
- PullInterval: 1000ms (1 second)
- PullBatchSize: 32
- MaxCachedMessageNum: 1000

### Migration Guide

#### From v1.x to v2.0

1. **Update AddProducer calls**:
   ```go
   // Old way
   config := ProducerConfig{MaxMessageSize: 1024}
   err = rocketMQ.AddProducer("topic", "group", config)
   
   // New way - using defaults
   err = rocketMQ.AddProducer("topic", "group", nil)
   
   // New way - with custom config
   config := &ProducerConfig{MaxMessageSize: 1024}
   err = rocketMQ.AddProducer("topic", "group", config)
   ```

2. **Update AddConsumer calls**:
   ```go
   // Old way
   config := ConsumerConfig{MaxReconsumeTimes: 10}
   err = rocketMQ.AddConsumer("topic", "group", handler, config)
   
   // New way - using defaults
   err = rocketMQ.AddConsumer("topic", "group", handler, nil)
   
   // New way - with custom config
   config := &ConsumerConfig{MaxReconsumeTimes: 10}
   err = rocketMQ.AddConsumer("topic", "group", handler, config)
   ```

3. **Optional: Set custom defaults**:
   ```go
   // Set your preferred defaults at application startup
   ezrocketmq.SetDefaultProducerConfig(ProducerConfig{
       MaxMessageSize: 8 * 1024 * 1024,
       SendMsgTimeout: 5000,
       RetryTimes: 5,
   })
   ```

## [1.x.x] - Previous versions

### Features
- Basic RocketMQ client wrapper
- Support for sync, async, and one-way message sending
- Support for clustering and broadcasting consume modes
- Message retry mechanism
- Delay message support
- Graceful startup and shutdown
- Authentication support via SetCredentials()
- Built-in logging functionality
- Producer and consumer state monitoring