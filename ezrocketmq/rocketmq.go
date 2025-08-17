package ezrocketmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/rs/zerolog"
)

type RocketMQ struct {
	config    Config
	producers map[string]rocketmq.Producer
	consumers map[string]rocketmq.PushConsumer
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	started   bool
}

// NewRocketMQ 创建新的RocketMQ实例
func NewRocketMQ(nameServers []string, instanceName, namespace string, enableLog bool) (*RocketMQ, error) {
	if err := ValidateNameServers(nameServers); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	r := &RocketMQ{
		config: Config{
			NameServers:  nameServers,
			InstanceName: instanceName,
			Namespace:    namespace,
			RetryTimes:   3,
			EnableLog:    enableLog,
			Producers:    make(map[string]ProducerConfig),
			Consumers:    make(map[string][]ConsumerConfig),
		},
		producers: make(map[string]rocketmq.Producer),
		consumers: make(map[string]rocketmq.PushConsumer),
		ctx:       ctx,
		cancel:    cancel,
	}

	return r, nil
}

// SetCredentials 设置认证信息
func (r *RocketMQ) SetCredentials(accessKey, secretKey, securityToken string) {
	r.config.Credentials = primitive.Credentials{
		AccessKey:     accessKey,
		SecretKey:     secretKey,
		SecurityToken: securityToken,
	}
}

// AddProducer 添加生产者配置
func (r *RocketMQ) AddProducer(topic, groupName string, config ProducerConfig) error {
	if err := ValidateTopicName(topic); err != nil {
		return err
	}
	if err := ValidateGroupName(groupName); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	config.Topic = topic
	config.GroupName = groupName

	// 设置默认值
	if config.MaxMessageSize <= 0 {
		config.MaxMessageSize = 4 * 1024 * 1024 // 4MB
	}
	if config.SendMsgTimeout <= 0 {
		config.SendMsgTimeout = 3000 // 3秒
	}
	if config.RetryTimes <= 0 {
		config.RetryTimes = 3
	}

	r.config.Producers[topic] = config
	return nil
}

// AddConsumer 添加消费者配置
func (r *RocketMQ) AddConsumer(topic, groupName string, handler MessageHandlerFunc, config ConsumerConfig) error {
	if err := ValidateTopicName(topic); err != nil {
		return err
	}
	if err := ValidateGroupName(groupName); err != nil {
		return err
	}
	if handler == nil {
		return &RocketMQError{
			Code:    -1,
			Message: "Message handler cannot be nil",
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	config.Topic = topic
	config.GroupName = groupName
	config.Handler = handler

	// 设置默认值
	if config.MaxReconsumeTimes <= 0 {
		config.MaxReconsumeTimes = 16
	}
	if config.ConsumeTimeout <= 0 {
		config.ConsumeTimeout = 15 // 15分钟
	}
	if config.PullInterval <= 0 {
		config.PullInterval = 1000 // 1秒
	}
	if config.PullBatchSize <= 0 {
		config.PullBatchSize = 32
	}
	if config.MaxCachedMessageNum <= 0 {
		config.MaxCachedMessageNum = 1000
	}

	r.config.Consumers[topic] = append(r.config.Consumers[topic], config)
	return nil
}

// Start 启动RocketMQ客户端
func (r *RocketMQ) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return &RocketMQError{
			Code:    -1,
			Message: "RocketMQ client already started",
		}
	}

	// 启动生产者
	for topic, producerConfig := range r.config.Producers {
		if err := r.startProducer(topic, producerConfig); err != nil {
			r.shutdown()
			return fmt.Errorf("failed to start producer for topic %s: %w", topic, err)
		}
	}

	// 启动消费者
	for topic, consumerConfigs := range r.config.Consumers {
		for i, consumerConfig := range consumerConfigs {
			if err := r.startConsumer(topic, consumerConfig, i); err != nil {
				r.shutdown()
				return fmt.Errorf("failed to start consumer for topic %s: %w", topic, err)
			}
		}
	}

	r.started = true
	Log(r.config.EnableLog, zerolog.InfoLevel, "RocketMQ client started successfully")
	return nil
}

// startProducer 启动生产者
func (r *RocketMQ) startProducer(topic string, config ProducerConfig) error {
	opts := []producer.Option{
		producer.WithNameServer(r.config.NameServers),
		producer.WithGroupName(config.GroupName),
		producer.WithInstanceName(r.config.InstanceName),
		producer.WithRetry(config.RetryTimes),
		producer.WithSendMsgTimeout(time.Duration(config.SendMsgTimeout) * time.Millisecond),
	}

	if r.config.Namespace != "" {
		opts = append(opts, producer.WithNamespace(r.config.Namespace))
	}

	if r.config.Credentials.AccessKey != "" {
		opts = append(opts, producer.WithCredentials(r.config.Credentials))
	}

	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		return err
	}

	if err := p.Start(); err != nil {
		return err
	}

	r.producers[topic] = p
	Log(r.config.EnableLog, zerolog.InfoLevel, "Producer started for topic: %s, group: %s", topic, config.GroupName)
	return nil
}

// startConsumer 启动消费者
func (r *RocketMQ) startConsumer(topic string, config ConsumerConfig, index int) error {
	consumerKey := fmt.Sprintf("%s_%d", topic, index)

	opts := []consumer.Option{
		consumer.WithNameServer(r.config.NameServers),
		consumer.WithGroupName(config.GroupName),

		consumer.WithMaxReconsumeTimes(config.MaxReconsumeTimes),
		consumer.WithConsumeTimeout(time.Duration(config.ConsumeTimeout) * time.Minute),
		consumer.WithPullInterval(time.Duration(config.PullInterval) * time.Millisecond),
		consumer.WithPullBatchSize(config.PullBatchSize),
	}

	if r.config.Namespace != "" {
		opts = append(opts, consumer.WithNamespace(r.config.Namespace))
	}

	if r.config.Credentials.AccessKey != "" {
		opts = append(opts, consumer.WithCredentials(r.config.Credentials))
	}

	// 设置消费模式
	switch config.ConsumeMode {
	case Broadcasting:
		opts = append(opts, consumer.WithConsumeMessageBatchMaxSize(1))
	default: // Clustering
		opts = append(opts, consumer.WithConsumeMessageBatchMaxSize(int(config.PullBatchSize)))
	}

	// 添加拦截器
	if len(config.Interceptors) > 0 {
		opts = append(opts, consumer.WithInterceptor(config.Interceptors...))
	}

	c, err := rocketmq.NewPushConsumer(opts...)
	if err != nil {
		return err
	}

	// 订阅主题
	tags := FormatTags(config.Tags)
	err = c.Subscribe(topic, consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: tags,
	}, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		result, err := config.Handler(ctx, msgs)
		if err != nil {
			Log(r.config.EnableLog, zerolog.ErrorLevel, "Message handler error: %v", err)
			return consumer.ConsumeRetryLater, err
		}

		switch result {
		case ConsumeSuccess:
			return consumer.ConsumeSuccess, nil
		case ConsumeRetryLater:
			return consumer.ConsumeRetryLater, nil
		default:
			return consumer.ConsumeRetryLater, nil
		}
	})

	if err != nil {
		return err
	}

	if err := c.Start(); err != nil {
		return err
	}

	r.consumers[consumerKey] = c
	Log(r.config.EnableLog, zerolog.InfoLevel, "Consumer started for topic: %s, group: %s, tags: %s", topic, config.GroupName, tags)
	return nil
}

// Send 发送消息
func (r *RocketMQ) Send(topic string, body []byte, opts *SendOptions) (*SendResult, error) {
	r.mu.RLock()
	producer, exists := r.producers[topic]
	r.mu.RUnlock()

	if !exists {
		return nil, &RocketMQError{
			Code:    -1,
			Message: fmt.Sprintf("Producer for topic %s not found", topic),
		}
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}

	if opts != nil {
		if opts.Tag != "" {
			msg.WithTag(opts.Tag)
		}
		if len(opts.Keys) > 0 {
			msg.WithKeys(opts.Keys)
		}
		if len(opts.Properties) > 0 {
			for k, v := range opts.Properties {
				msg.WithProperty(k, v)
			}
		}
		if opts.DelayLevel > 0 {
			msg.WithDelayTimeLevel(opts.DelayLevel)
		}
	}

	ctx := r.ctx
	if opts != nil && opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(r.ctx, time.Duration(opts.Timeout)*time.Millisecond)
		defer cancel()
	}

	result, err := producer.SendSync(ctx, msg)
	if err != nil {
		return nil, &RocketMQError{
			Code:    -1,
			Message: "Failed to send message",
			Err:     err,
		}
	}

	return &SendResult{
		MessageID:   result.MsgID,
		QueueID:     int(result.MessageQueue.QueueId),
		QueueOffset: result.QueueOffset,
		Status:      result.Status,
		MsgExt:      nil,
	}, nil
}

// SendAsync 异步发送消息
func (r *RocketMQ) SendAsync(topic string, body []byte, opts *SendOptions, callback func(*SendResult, error)) error {
	r.mu.RLock()
	producer, exists := r.producers[topic]
	r.mu.RUnlock()

	if !exists {
		return &RocketMQError{
			Code:    -1,
			Message: fmt.Sprintf("Producer for topic %s not found", topic),
		}
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}

	if opts != nil {
		if opts.Tag != "" {
			msg.WithTag(opts.Tag)
		}
		if len(opts.Keys) > 0 {
			msg.WithKeys(opts.Keys)
		}
		if len(opts.Properties) > 0 {
			for k, v := range opts.Properties {
				msg.WithProperty(k, v)
			}
		}
		if opts.DelayLevel > 0 {
			msg.WithDelayTimeLevel(opts.DelayLevel)
		}
	}

	ctx := r.ctx
	if opts != nil && opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(r.ctx, time.Duration(opts.Timeout)*time.Millisecond)
		defer cancel()
	}

	return producer.SendAsync(ctx, func(ctx context.Context, result *primitive.SendResult, err error) {
		if callback != nil {
			if err != nil {
				callback(nil, &RocketMQError{
					Code:    -1,
					Message: "Failed to send message async",
					Err:     err,
				})
			} else {
				callback(&SendResult{
					MessageID:   result.MsgID,
					QueueID:     int(result.MessageQueue.QueueId),
					QueueOffset: result.QueueOffset,
					Status:      result.Status,
					MsgExt:      nil,
				}, nil)
			}
		}
	}, msg)
}

// SendOneWay 单向发送消息（不关心发送结果）
func (r *RocketMQ) SendOneWay(topic string, body []byte, opts *SendOptions) error {
	r.mu.RLock()
	producer, exists := r.producers[topic]
	r.mu.RUnlock()

	if !exists {
		return &RocketMQError{
			Code:    -1,
			Message: fmt.Sprintf("Producer for topic %s not found", topic),
		}
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}

	if opts != nil {
		if opts.Tag != "" {
			msg.WithTag(opts.Tag)
		}
		if len(opts.Keys) > 0 {
			msg.WithKeys(opts.Keys)
		}
		if len(opts.Properties) > 0 {
			for k, v := range opts.Properties {
				msg.WithProperty(k, v)
			}
		}
		if opts.DelayLevel > 0 {
			msg.WithDelayTimeLevel(opts.DelayLevel)
		}
	}

	ctx := r.ctx
	if opts != nil && opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(r.ctx, time.Duration(opts.Timeout)*time.Millisecond)
		defer cancel()
	}

	err := producer.SendOneWay(ctx, msg)
	if err != nil {
		return &RocketMQError{
			Code:    -1,
			Message: "Failed to send message one way",
			Err:     err,
		}
	}

	return nil
}

// Stop 停止RocketMQ客户端
func (r *RocketMQ) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return
	}

	r.shutdown()
	r.started = false
	Log(r.config.EnableLog, zerolog.InfoLevel, "RocketMQ client stopped")
}

// shutdown 内部关闭方法
func (r *RocketMQ) shutdown() {
	// 取消context
	r.cancel()

	// 关闭消费者
	for key, consumer := range r.consumers {
		if err := consumer.Shutdown(); err != nil {
			Log(r.config.EnableLog, zerolog.ErrorLevel, "Failed to shutdown consumer %s: %v", key, err)
		} else {
			Log(r.config.EnableLog, zerolog.InfoLevel, "Consumer %s shutdown successfully", key)
		}
	}

	// 关闭生产者
	for topic, producer := range r.producers {
		if err := producer.Shutdown(); err != nil {
			Log(r.config.EnableLog, zerolog.ErrorLevel, "Failed to shutdown producer for topic %s: %v", topic, err)
		} else {
			Log(r.config.EnableLog, zerolog.InfoLevel, "Producer for topic %s shutdown successfully", topic)
		}
	}

	// 等待所有goroutine完成
	r.wg.Wait()

	// 清空maps
	r.producers = make(map[string]rocketmq.Producer)
	r.consumers = make(map[string]rocketmq.PushConsumer)
}

// GetProducerState 获取生产者状态
func (r *RocketMQ) GetProducerState(topic string) ProducerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.producers[topic]; exists && r.started {
		return ProducerStateStarted
	}
	if _, exists := r.config.Producers[topic]; exists {
		return ProducerStateCreated
	}
	return ProducerStateStopped
}

// GetConsumerState 获取消费者状态
func (r *RocketMQ) GetConsumerState(topic string) ConsumerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for key := range r.consumers {
		if key == topic || (len(key) > len(topic) && key[:len(topic)] == topic) {
			if r.started {
				return ConsumerStateStarted
			}
			return ConsumerStateCreated
		}
	}
	return ConsumerStateStopped
}

// IsStarted 检查客户端是否已启动
func (r *RocketMQ) IsStarted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.started
}
