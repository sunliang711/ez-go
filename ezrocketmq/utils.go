package ezrocketmq

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Log 如果enableLog为true，则打印日志
func Log(enable bool, level zerolog.Level, format string, args ...any) {
	if !enable {
		return
	}

	switch level {
	case zerolog.TraceLevel:
		log.Trace().Msgf(format, args...)
	case zerolog.DebugLevel:
		log.Debug().Msgf(format, args...)
	case zerolog.InfoLevel:
		log.Info().Msgf(format, args...)
	case zerolog.WarnLevel:
		log.Warn().Msgf(format, args...)
	case zerolog.ErrorLevel:
		log.Error().Msgf(format, args...)
	case zerolog.FatalLevel:
		log.Fatal().Msgf(format, args...)
	}
}

// ConvertConsumeFromWhere 转换消费起始位置
func ConvertConsumeFromWhere(where ConsumeFromWhere) string {
	switch where {
	case ConsumeFromFirstOffset:
		return "CONSUME_FROM_FIRST_OFFSET"
	case ConsumeFromLastOffset:
		return "CONSUME_FROM_LAST_OFFSET"
	case ConsumeFromTimestamp:
		return "CONSUME_FROM_TIMESTAMP"
	case ConsumeFromStoredOffset:
		return "CONSUME_FROM_STORED_OFFSET"
	default:
		return "CONSUME_FROM_LAST_OFFSET"
	}
}

// FormatTags 格式化标签列表为字符串
func FormatTags(tags []string) string {
	if len(tags) == 0 {
		return "*"
	}

	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += " || "
		}
		result += tag
	}
	return result
}

// ValidateNameServers 验证NameServer地址列表
func ValidateNameServers(nameServers []string) error {
	if len(nameServers) == 0 {
		return &RocketMQError{
			Code:    -1,
			Message: "NameServers cannot be empty",
		}
	}
	return nil
}

// ValidateTopicName 验证Topic名称
func ValidateTopicName(topic string) error {
	if topic == "" {
		return &RocketMQError{
			Code:    -1,
			Message: "Topic name cannot be empty",
		}
	}
	return nil
}

// ValidateGroupName 验证组名
func ValidateGroupName(groupName string) error {
	if groupName == "" {
		return &RocketMQError{
			Code:    -1,
			Message: "Group name cannot be empty",
		}
	}
	return nil
}
