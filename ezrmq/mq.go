package ezrmq

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/makasim/amqpextra"
	"github.com/makasim/amqpextra/consumer"
	"github.com/makasim/amqpextra/publisher"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	config     Config
	dialer     *amqpextra.Dialer
	publisher  *publisher.Publisher
	consumers  []*consumer.Consumer
	wg         sync.WaitGroup
	consumeMux sync.Mutex

	logger *log.Logger

	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewRabbitMQ creates a new RabbitMQ instance
// caCertBytes如果为空，则不使用服务端TLS
// clientCert和clientKey如果为空，则不使用客户端TLS
func NewRabbitMQ(url string, reconnect int, caCertBytes, clientCert, clientKey []byte) (*RabbitMQ, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &RabbitMQ{
		config: Config{
			URL:          url,
			ReconnectSec: reconnect,
			CaCertBytes:  caCertBytes,
			ClientCert:   clientCert,
			ClientKey:    clientKey,
			Consumers:    make(map[string][]ConsumerConfig),
			Producers:    make(map[string]ProducerConfig),
		},
		logger:     log.New(os.Stdout, "|RMQ| ", log.LstdFlags),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	return r, nil
}

func (r *RabbitMQ) AddConsumer(exchangeName string, topic string, handler MessageHandlerFunc, queueOptions QueueOptions, consumeOptions ConsumeOptions) error {
	config := ConsumerConfig{
		Handler:        handler,
		Topic:          topic,
		QueueOptions:   queueOptions,
		ConsumeOptions: consumeOptions,
	}

	if handler == nil {
		return fmt.Errorf("exchange %s handler is nil", exchangeName)
	}

	r.config.Consumers[exchangeName] = append(r.config.Consumers[exchangeName], config)
	return nil
}

func (r *RabbitMQ) AddProducer(exchangeName string, exchangeOptions ExchangeOptions) error {
	exchange := ProducerConfig{
		ExchangeOptions: exchangeOptions,
	}

	r.config.Producers[exchangeName] = exchange
	return nil
}

func (r *RabbitMQ) Connect() error {
	var err error
	var options []amqpextra.Option

	// 创建状态通知通道 - 使用缓冲通道
	dialerStateCh := make(chan amqpextra.State, 10)
	options = append(options, amqpextra.WithNotify(dialerStateCh))

	// 设置连接重试
	options = append(options, amqpextra.WithConnectionProperties(amqp.Table{
		"connection_name": "ezrmq",
	}))

	options = append(options, amqpextra.WithRetryPeriod(time.Duration(r.config.ReconnectSec)*time.Second))
	options = append(options, amqpextra.WithURL(r.config.URL))
	options = append(options, amqpextra.WithContext(r.ctx))
	options = append(options, amqpextra.WithLogger(r.logger))

	// 如果使用TLS，设置TLS配置
	if len(r.config.CaCertBytes) > 0 {
		r.logger.Printf("config server ca certificate")
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(r.config.CaCertBytes)

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}

		// 如果clientCert和clientKey非空，则使用客户端TLS
		if len(r.config.ClientCert) > 0 && len(r.config.ClientKey) > 0 {
			r.logger.Printf("config client certificate")
			clientCert, err := tls.X509KeyPair(r.config.ClientCert, r.config.ClientKey)
			if err != nil {
				return err
			}
			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}

		options = append(options, amqpextra.WithTLS(tlsConfig))
	}

	// 初始化dialer
	r.dialer, err = amqpextra.NewDialer(options...)
	if err != nil {
		return fmt.Errorf("failed to create dialer: %w", err)
	}

	// 监听dialer状态变化
	go r.monitorDialerState(dialerStateCh)

	// 初始化publisher
	publisherStateCh := make(chan publisher.State, 10)
	r.publisher, err = r.dialer.Publisher(
		publisher.WithNotify(publisherStateCh),
	)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	// 监听publisher状态
	go r.monitorPublisherState(publisherStateCh)

	// 声明producer exchanges
	for exchangeName, producerConfig := range r.config.Producers {
		r.logger.Printf("Declare exchange: %s", exchangeName)
		err = r.declareExchange(exchangeName, producerConfig.ExchangeOptions)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", exchangeName, err)
		}
	}

	// 初始化consumers
	for exchangeName, consumerConfigs := range r.config.Consumers {
		for _, consumerConfig := range consumerConfigs {
			err = r.setupConsumer(exchangeName, &consumerConfig)
			if err != nil {
				return fmt.Errorf("failed to setup consumer for exchange %s: %w", exchangeName, err)
			}
		}
	}

	return nil
}

// 监控Dialer状态
func (r *RabbitMQ) monitorDialerState(stateCh <-chan amqpextra.State) {
	for state := range stateCh {
		if state.Ready != nil {
			r.logger.Printf("RabbitMQ连接就绪")
		}
		if state.Unready != nil {
			r.logger.Printf("RabbitMQ连接断开: %v", state.Unready.Err)
		}
	}
	r.logger.Printf("Dialer状态监控结束")
}

// 监控Publisher状态
func (r *RabbitMQ) monitorPublisherState(stateCh <-chan publisher.State) {
	for state := range stateCh {
		if state.Ready != nil {
			r.logger.Printf("Publisher就绪")
		}
		if state.Unready != nil {
			r.logger.Printf("Publisher断开: %v", state.Unready.Err)
		}
	}
	r.logger.Printf("Publisher状态监控结束")
}

// declareExchange declares an exchange
func (r *RabbitMQ) declareExchange(exchangeName string, options ExchangeOptions) error {
	conn, err := r.dialer.Connection(r.ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.ExchangeDeclare(
		exchangeName,      // name
		options.Type,      // type
		options.Durable,   // durable
		options.AutoDel,   // auto-deleted
		options.Internal,  // internal
		options.NoWait,    // no-wait
		options.Arguments, // arguments
	)
}

// 添加手动绑定队列和交换机的方法
func (r *RabbitMQ) bindQueueToExchange(queueName, exchangeName, routingKey string, noWait bool, arguments amqp.Table) error {
	if arguments == nil {
		arguments = amqp.Table{}
	}

	conn, err := r.dialer.Connection(r.ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	r.logger.Printf("Binding queue: %s to exchange: %s with topic: %s", queueName, exchangeName, routingKey)
	return ch.QueueBind(
		queueName,    // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		noWait,       // no-wait
		arguments,    // arguments
	)
}

// setupConsumer initializes a consumer
func (r *RabbitMQ) setupConsumer(exchangeName string, consumerConfig *ConsumerConfig) error {
	r.consumeMux.Lock()
	defer r.consumeMux.Unlock()

	// 首先需要确保队列存在
	conn, err := r.dialer.Connection(r.ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// 声明队列
	r.logger.Printf("Declaring queue: %s", consumerConfig.QueueOptions.Name)
	_, err = ch.QueueDeclare(
		consumerConfig.QueueOptions.Name,
		consumerConfig.QueueOptions.Durable,
		consumerConfig.QueueOptions.AutoDel,
		consumerConfig.QueueOptions.Exclusive,
		consumerConfig.QueueOptions.NoWait,
		consumerConfig.QueueOptions.Arguments,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机
	err = r.bindQueueToExchange(consumerConfig.QueueOptions.Name, exchangeName, consumerConfig.Topic, consumerConfig.QueueOptions.BindNoWait, consumerConfig.QueueOptions.BindArgs)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// 创建消息处理函数的handler
	handler := consumer.HandlerFunc(func(ctx context.Context, msg amqp.Delivery) interface{} {
		consumerConfig.Handler(msg)
		return nil
	})

	// 创建状态通知通道 - 使用缓冲通道
	consumerStateCh := make(chan consumer.State, 10)

	// 创建consumer options - 这里不再需要声明队列和绑定的选项，因为我们已经手动完成了
	opts := []consumer.Option{
		consumer.WithContext(r.ctx),
		consumer.WithQueue(consumerConfig.QueueOptions.Name),
		// 不需要WithExchange，因为我们已经手动绑定了
		consumer.WithConsumeArgs(
			"", // consumer name (auto-generated)
			consumerConfig.ConsumeOptions.AutoAck,
			consumerConfig.ConsumeOptions.Exclusive,
			consumerConfig.ConsumeOptions.NoLocal,
			consumerConfig.ConsumeOptions.NoWait,
			consumerConfig.ConsumeOptions.Arguments,
		),
		consumer.WithHandler(handler),
		consumer.WithNotify(consumerStateCh),
	}

	// 创建consumer
	c, err := r.dialer.Consumer(opts...)
	if err != nil {
		return err
	}

	// 监控consumer状态
	queueName := consumerConfig.QueueOptions.Name
	go r.monitorConsumerState(consumerStateCh, queueName)

	// 存储consumer便于关闭
	r.consumers = append(r.consumers, c)

	// 启动consumer
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		<-c.NotifyClosed()
		r.logger.Printf("Consumer for queue %s has been closed", consumerConfig.QueueOptions.Name)
	}()

	return nil
}

// 监控Consumer状态
func (r *RabbitMQ) monitorConsumerState(stateCh <-chan consumer.State, queueName string) {
	for state := range stateCh {
		if state.Ready != nil {
			r.logger.Printf("Consumer队列 %s 已就绪", queueName)
		}
		if state.Unready != nil {
			r.logger.Printf("Consumer队列 %s 断开连接: %v", queueName, state.Unready.Err)
		}
	}
	r.logger.Printf("Consumer状态监控结束: %s", queueName)
}

// Publish publishes a message to the specified exchange with the given routing key
func (r *RabbitMQ) Publish(exchange, routingKey string, body []byte) error {
	return r.publisher.Publish(publisher.Message{
		Context:   r.ctx,
		Exchange:  exchange,
		Key:       routingKey,
		Mandatory: false,
		Immediate: false,
		Publishing: amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		},
	})
}

// Close gracefully closes the RabbitMQ connection
func (r *RabbitMQ) Close() {
	r.logger.Println("Closing RabbitMQ connection...")
	r.cancelFunc()

	// Close publisher
	if r.publisher != nil {
		r.publisher.Close()
	}

	// Close dialer after all consumers have stopped
	r.logger.Println("Waiting for all consumers to stop...")
	r.wg.Wait()
	r.logger.Println("All consumers stopped")

	if r.dialer != nil {
		r.dialer.Close()
	}
}
