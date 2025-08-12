package ezcamunda

import (
	"fmt"
	"reflect"
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/rs/zerolog/log"
)

// AutoRegisterOptions 自动注册选项
type AutoRegisterOptions struct {
	Prefix        string        // JobType 前缀
	Concurrency   int           // 默认并发数
	MaxJobActives int           // 默认最大活跃任务数
	Timeout       time.Duration // 默认超时时间
	PollInterval  time.Duration // 默认轮询间隔
}

// DefaultAutoRegisterOptions 返回默认的自动注册选项
func DefaultAutoRegisterOptions() *AutoRegisterOptions {
	return &AutoRegisterOptions{
		Prefix:        "",
		Concurrency:   5,
		MaxJobActives: 5,
		Timeout:       10 * time.Second,
		PollInterval:  10 * time.Second,
	}
}

// AutoRegister 自动注册结构体中符合 JobHandler 签名的方法
// handler: 包含处理方法的结构体指针
// options: 注册选项，如果为 nil 则使用默认选项
func (cli *CamundaClient) AutoRegister(handler any, options *AutoRegisterOptions) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	// 使用默认选项
	if options == nil {
		options = DefaultAutoRegisterOptions()
	}

	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	// 确保传入的是指针
	if handlerType.Kind() != reflect.Ptr {
		return fmt.Errorf("handler must be a pointer to struct, got %s", handlerType.Kind())
	}

	// 获取结构体类型
	structType := handlerType.Elem()
	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("handler must be a pointer to struct, got pointer to %s", structType.Kind())
	}

	structName := structType.Name()
	registeredCount := 0

	// 遍历所有方法
	for i := 0; i < handlerType.NumMethod(); i++ {
		method := handlerType.Method(i)
		methodType := method.Type

		// 检查方法签名是否匹配 JobHandler
		if isJobHandlerMethod(methodType) {
			// 创建 JobHandler
			jobHandler := createJobHandler(handlerValue, method)

			// 生成 JobType 和 WorkerName
			jobType := generateJobType(options.Prefix, structName, method.Name)
			workerName := generateWorkerName(structName, method.Name)

			// 创建 CamundaWorker
			worker := NewCamundaWorker(
				jobType,
				workerName,
				jobHandler,
				options.Concurrency,
				options.MaxJobActives,
				options.Timeout,
				options.PollInterval,
			)

			// 添加到 workers 列表
			cli.AddWorkers(*worker)
			registeredCount++

			log.Info().
				Str("jobType", jobType).
				Str("workerName", workerName).
				Str("method", method.Name).
				Msg("Auto registered worker")
		}
	}

	if registeredCount == 0 {
		log.Warn().
			Str("struct", structName).
			Msg("No methods matching JobHandler signature found")
	} else {
		log.Info().
			Str("struct", structName).
			Int("count", registeredCount).
			Msg("Auto registration completed")
	}

	return nil
}

// isJobHandlerMethod 检查方法签名是否匹配 JobHandler
// JobHandler: func(client JobClient, job entities.Job)
func isJobHandlerMethod(methodType reflect.Type) bool {
	// 检查参数数量（包括 receiver）
	// receiver + client + job = 3
	if methodType.NumIn() != 3 {
		return false
	}

	// 检查返回值数量（应该为 0）
	if methodType.NumOut() != 0 {
		return false
	}

	// 检查第二个参数是否实现了 worker.JobClient 接口
	clientType := methodType.In(1)
	jobClientInterface := reflect.TypeOf((*worker.JobClient)(nil)).Elem()
	if !clientType.Implements(jobClientInterface) {
		return false
	}

	// 检查第三个参数是否是 entities.Job
	jobType := methodType.In(2)
	expectedJobType := reflect.TypeOf(entities.Job{})
	if jobType != expectedJobType {
		return false
	}

	return true
}

// createJobHandler 创建 JobHandler 函数
func createJobHandler(receiver reflect.Value, method reflect.Method) worker.JobHandler {
	return func(client worker.JobClient, job entities.Job) {
		// 调用方法
		method.Func.Call([]reflect.Value{
			receiver,
			reflect.ValueOf(client),
			reflect.ValueOf(job),
		})
	}
}

// generateJobType 生成 JobType
func generateJobType(prefix, structName, methodName string) string {
	// parts := []string{}

	// if prefix != "" {
	// 	parts = append(parts, prefix)
	// }

	// // 将结构体名和方法名转换为 kebab-case
	// structPart := toKebabCase(structName)
	// methodPart := toKebabCase(methodName)

	// parts = append(parts, structPart, methodPart)

	// return strings.Join(parts, "-")
	return methodName
}

// generateWorkerName 生成 WorkerName
func generateWorkerName(structName, methodName string) string {
	return fmt.Sprintf("%s.%s", structName, methodName)
}

// toKebabCase 将 CamelCase 转换为 kebab-case
func toKebabCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '-')
		}
		result = append(result, toLower(r))
	}
	return string(result)
}

// toLower 将大写字母转换为小写
func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// AutoRegisterWithPrefix 使用指定前缀自动注册
func (cli *CamundaClient) AutoRegisterWithPrefix(handler interface{}, prefix string) error {
	options := DefaultAutoRegisterOptions()
	options.Prefix = prefix
	return cli.AutoRegister(handler, options)
}

// BatchAutoRegister 批量自动注册多个处理器
func (cli *CamundaClient) BatchAutoRegister(handlers []interface{}, options *AutoRegisterOptions) error {
	for _, handler := range handlers {
		if err := cli.AutoRegister(handler, options); err != nil {
			return fmt.Errorf("failed to auto register handler: %w", err)
		}
	}
	return nil
}
