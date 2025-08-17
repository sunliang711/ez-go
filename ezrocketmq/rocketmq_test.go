package ezrocketmq

import (
	"context"
	"testing"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

func TestNewRocketMQ(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	if rocketMQ == nil {
		t.Fatal("RocketMQ instance is nil")
	}

	if len(rocketMQ.config.NameServers) != 1 {
		t.Errorf("Expected 1 name server, got %d", len(rocketMQ.config.NameServers))
	}

	if rocketMQ.config.InstanceName != "test-instance" {
		t.Errorf("Expected instance name 'test-instance', got '%s'", rocketMQ.config.InstanceName)
	}
}

func TestNewRocketMQWithEmptyNameServers(t *testing.T) {
	nameServers := []string{}
	_, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err == nil {
		t.Fatal("Expected error for empty name servers, got nil")
	}

	rocketErr, ok := err.(*RocketMQError)
	if !ok {
		t.Fatalf("Expected RocketMQError, got %T", err)
	}

	if rocketErr.Code != -1 {
		t.Errorf("Expected error code -1, got %d", rocketErr.Code)
	}
}

func TestSetCredentials(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	accessKey := "test-access-key"
	secretKey := "test-secret-key"
	securityToken := "test-token"

	rocketMQ.SetCredentials(accessKey, secretKey, securityToken)

	if rocketMQ.config.Credentials.AccessKey != accessKey {
		t.Errorf("Expected access key '%s', got '%s'", accessKey, rocketMQ.config.Credentials.AccessKey)
	}

	if rocketMQ.config.Credentials.SecretKey != secretKey {
		t.Errorf("Expected secret key '%s', got '%s'", secretKey, rocketMQ.config.Credentials.SecretKey)
	}

	if rocketMQ.config.Credentials.SecurityToken != securityToken {
		t.Errorf("Expected security token '%s', got '%s'", securityToken, rocketMQ.config.Credentials.SecurityToken)
	}
}

func TestAddProducer(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	topic := "test-topic"
	groupName := "test-producer-group"
	config := ProducerConfig{
		MaxMessageSize: 1024,
		SendMsgTimeout: 5000,
		RetryTimes:     5,
		Tags:           []string{"tag1", "tag2"},
	}

	err = rocketMQ.AddProducer(topic, groupName, config)
	if err != nil {
		t.Fatalf("Failed to add producer: %v", err)
	}

	producerConfig, exists := rocketMQ.config.Producers[topic]
	if !exists {
		t.Fatal("Producer config not found")
	}

	if producerConfig.Topic != topic {
		t.Errorf("Expected topic '%s', got '%s'", topic, producerConfig.Topic)
	}

	if producerConfig.GroupName != groupName {
		t.Errorf("Expected group name '%s', got '%s'", groupName, producerConfig.GroupName)
	}

	if producerConfig.MaxMessageSize != 1024 {
		t.Errorf("Expected max message size 1024, got %d", producerConfig.MaxMessageSize)
	}
}

func TestAddProducerWithDefaults(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	topic := "test-topic"
	groupName := "test-producer-group"
	config := ProducerConfig{} // Empty config to test defaults

	err = rocketMQ.AddProducer(topic, groupName, config)
	if err != nil {
		t.Fatalf("Failed to add producer: %v", err)
	}

	producerConfig := rocketMQ.config.Producers[topic]

	if producerConfig.MaxMessageSize != 4*1024*1024 {
		t.Errorf("Expected default max message size %d, got %d", 4*1024*1024, producerConfig.MaxMessageSize)
	}

	if producerConfig.SendMsgTimeout != 3000 {
		t.Errorf("Expected default send timeout 3000, got %d", producerConfig.SendMsgTimeout)
	}

	if producerConfig.RetryTimes != 3 {
		t.Errorf("Expected default retry times 3, got %d", producerConfig.RetryTimes)
	}
}

func TestAddConsumer(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	topic := "test-topic"
	groupName := "test-consumer-group"
	handler := func(ctx context.Context, msgs []*primitive.MessageExt) (ConsumeResult, error) {
		return ConsumeSuccess, nil
	}
	config := ConsumerConfig{
		Tags:                []string{"tag1"},
		ConsumeFromWhere:    ConsumeFromLastOffset,
		ConsumeMode:         Clustering,
		MaxReconsumeTimes:   10,
		ConsumeTimeout:      20,
		PullInterval:        2000,
		PullBatchSize:       64,
		MaxCachedMessageNum: 2000,
	}

	err = rocketMQ.AddConsumer(topic, groupName, handler, config)
	if err != nil {
		t.Fatalf("Failed to add consumer: %v", err)
	}

	consumerConfigs, exists := rocketMQ.config.Consumers[topic]
	if !exists {
		t.Fatal("Consumer config not found")
	}

	if len(consumerConfigs) != 1 {
		t.Fatalf("Expected 1 consumer config, got %d", len(consumerConfigs))
	}

	consumerConfig := consumerConfigs[0]

	if consumerConfig.Topic != topic {
		t.Errorf("Expected topic '%s', got '%s'", topic, consumerConfig.Topic)
	}

	if consumerConfig.GroupName != groupName {
		t.Errorf("Expected group name '%s', got '%s'", groupName, consumerConfig.GroupName)
	}

	if len(consumerConfig.Tags) != 1 || consumerConfig.Tags[0] != "tag1" {
		t.Errorf("Expected tags ['tag1'], got %v", consumerConfig.Tags)
	}
}

func TestAddConsumerWithDefaults(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	topic := "test-topic"
	groupName := "test-consumer-group"
	handler := func(ctx context.Context, msgs []*primitive.MessageExt) (ConsumeResult, error) {
		return ConsumeSuccess, nil
	}
	config := ConsumerConfig{} // Empty config to test defaults

	err = rocketMQ.AddConsumer(topic, groupName, handler, config)
	if err != nil {
		t.Fatalf("Failed to add consumer: %v", err)
	}

	consumerConfig := rocketMQ.config.Consumers[topic][0]

	if consumerConfig.MaxReconsumeTimes != 16 {
		t.Errorf("Expected default max reconsume times 16, got %d", consumerConfig.MaxReconsumeTimes)
	}

	if consumerConfig.ConsumeTimeout != 15 {
		t.Errorf("Expected default consume timeout 15, got %d", consumerConfig.ConsumeTimeout)
	}

	if consumerConfig.PullInterval != 1000 {
		t.Errorf("Expected default pull interval 1000, got %d", consumerConfig.PullInterval)
	}

	if consumerConfig.PullBatchSize != 32 {
		t.Errorf("Expected default pull batch size 32, got %d", consumerConfig.PullBatchSize)
	}

	if consumerConfig.MaxCachedMessageNum != 1000 {
		t.Errorf("Expected default max cached message num 1000, got %d", consumerConfig.MaxCachedMessageNum)
	}
}

func TestAddConsumerWithNilHandler(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	topic := "test-topic"
	groupName := "test-consumer-group"
	config := ConsumerConfig{}

	err = rocketMQ.AddConsumer(topic, groupName, nil, config)
	if err == nil {
		t.Fatal("Expected error for nil handler, got nil")
	}

	rocketErr, ok := err.(*RocketMQError)
	if !ok {
		t.Fatalf("Expected RocketMQError, got %T", err)
	}

	if rocketErr.Code != -1 {
		t.Errorf("Expected error code -1, got %d", rocketErr.Code)
	}
}

func TestValidateTopicName(t *testing.T) {
	tests := []struct {
		topic     string
		expectErr bool
	}{
		{"valid-topic", false},
		{"", true},
		{"test_topic", false},
		{"TopicWithNumbers123", false},
	}

	for _, test := range tests {
		err := ValidateTopicName(test.topic)
		if test.expectErr && err == nil {
			t.Errorf("Expected error for topic '%s', got nil", test.topic)
		}
		if !test.expectErr && err != nil {
			t.Errorf("Expected no error for topic '%s', got %v", test.topic, err)
		}
	}
}

func TestValidateGroupName(t *testing.T) {
	tests := []struct {
		groupName string
		expectErr bool
	}{
		{"valid-group", false},
		{"", true},
		{"test_group", false},
		{"GroupWithNumbers123", false},
	}

	for _, test := range tests {
		err := ValidateGroupName(test.groupName)
		if test.expectErr && err == nil {
			t.Errorf("Expected error for group name '%s', got nil", test.groupName)
		}
		if !test.expectErr && err != nil {
			t.Errorf("Expected no error for group name '%s', got %v", test.groupName, err)
		}
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		tags     []string
		expected string
	}{
		{[]string{}, "*"},
		{[]string{"tag1"}, "tag1"},
		{[]string{"tag1", "tag2"}, "tag1 || tag2"},
		{[]string{"tag1", "tag2", "tag3"}, "tag1 || tag2 || tag3"},
	}

	for _, test := range tests {
		result := FormatTags(test.tags)
		if result != test.expected {
			t.Errorf("Expected '%s', got '%s' for tags %v", test.expected, result, test.tags)
		}
	}
}

func TestGetProducerState(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	topic := "test-topic"

	// Initially should be stopped
	state := rocketMQ.GetProducerState(topic)
	if state != ProducerStateStopped {
		t.Errorf("Expected ProducerStateStopped, got %v", state)
	}

	// After adding producer, should be created
	config := ProducerConfig{}
	err = rocketMQ.AddProducer(topic, "test-group", config)
	if err != nil {
		t.Fatalf("Failed to add producer: %v", err)
	}

	state = rocketMQ.GetProducerState(topic)
	if state != ProducerStateCreated {
		t.Errorf("Expected ProducerStateCreated, got %v", state)
	}
}

func TestIsStarted(t *testing.T) {
	nameServers := []string{"127.0.0.1:9876"}
	rocketMQ, err := NewRocketMQ(nameServers, "test-instance", "", false)
	if err != nil {
		t.Fatalf("Failed to create RocketMQ instance: %v", err)
	}

	// Initially should not be started
	if rocketMQ.IsStarted() {
		t.Error("Expected IsStarted to be false initially")
	}
}

func TestRocketMQError(t *testing.T) {
	err := &RocketMQError{
		Code:    100,
		Message: "test error",
	}

	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}

	innerErr := &RocketMQError{
		Code:    200,
		Message: "outer error",
		Err:     err,
	}

	expected := "outer error: test error"
	if innerErr.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, innerErr.Error())
	}
}
