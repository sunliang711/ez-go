package ezrocketmq

import (
	"context"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type Config struct {
	NameServers  []string // RocketMQ NameServer地址列表
	Credentials  primitive.Credentials
	InstanceName string // 实例名称，用于区分不同的客户端
	Namespace    string // 命名空间
	RetryTimes   int    // 重试次数
	EnableLog    bool   // 是否启用日志

	Producers map[string]ProducerConfig   // 用于生产消息, key为topic name, value为配置信息
	Consumers map[string][]ConsumerConfig // 用于消费消息, key为topic name, value为配置信息列表
}

type ProducerConfig struct {
	Topic          string            // Topic名称
	GroupName      string            // 生产者组名
	MaxMessageSize int               // 最大消息大小
	SendMsgTimeout int               // 发送超时时间(毫秒)
	RetryTimes     int               // 重试次数
	Tags           []string          // 支持的标签列表
	Properties     map[string]string // 自定义属性
}

type ConsumerConfig struct {
	Topic               string                  // Topic名称
	GroupName           string                  // 消费者组名
	Tags                []string                // 订阅的标签，如果为空则订阅所有
	Handler             MessageHandlerFunc      // 消息处理函数
	ConsumeFromWhere    ConsumeFromWhere        // 消费起始位置
	ConsumeMode         ConsumeMode             // 消费模式
	MaxReconsumeTimes   int32                   // 最大重消费次数
	ConsumeTimeout      int64                   // 消费超时时间(分钟)
	PullInterval        int64                   // 拉取间隔(毫秒)
	PullBatchSize       int32                   // 批量拉取大小
	MaxCachedMessageNum int32                   // 最大缓存消息数量
	Properties          map[string]string       // 自定义属性
	Interceptors        []primitive.Interceptor // 拦截器
}

// 消费起始位置
type ConsumeFromWhere int

const (
	ConsumeFromFirstOffset  ConsumeFromWhere = iota // 从队列头开始消费
	ConsumeFromLastOffset                           // 从队列尾开始消费
	ConsumeFromTimestamp                            // 从指定时间点开始消费
	ConsumeFromStoredOffset                         // 从存储的消费位点开始消费
)

// 消费模式
type ConsumeMode int

const (
	Clustering   ConsumeMode = iota // 集群消费模式
	Broadcasting                    // 广播消费模式
)

// 消息处理函数类型
type MessageHandlerFunc func(ctx context.Context, msgs []*primitive.MessageExt) (ConsumeResult, error)

// 消费结果
type ConsumeResult int

const (
	ConsumeSuccess    ConsumeResult = iota // 消费成功
	ConsumeRetryLater                      // 稍后重试
)

// 发送消息选项
type SendOptions struct {
	Tag        string            // 消息标签
	Keys       []string          // 消息关键字
	Properties map[string]string // 自定义属性
	DelayLevel int               // 延时级别 (1-18, 对应不同的延时时间)
	Timeout    int64             // 发送超时时间(毫秒)
}

// 消息发送结果
type SendResult struct {
	MessageID   string                // 消息ID
	QueueID     int                   // 队列ID
	QueueOffset int64                 // 队列偏移量
	Status      primitive.SendStatus  // 发送状态
	MsgExt      *primitive.MessageExt // 消息扩展信息
}

// 错误类型
type RocketMQError struct {
	Code    int    // 错误码
	Message string // 错误信息
	Err     error  // 原始错误
}

func (e *RocketMQError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// 生产者状态
type ProducerState int

const (
	ProducerStateCreated ProducerState = iota
	ProducerStateStarted
	ProducerStateStopped
)

// 消费者状态
type ConsumerState int

const (
	ConsumerStateCreated ConsumerState = iota
	ConsumerStateStarted
	ConsumerStateStopped
)
