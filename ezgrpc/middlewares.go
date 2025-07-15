package ezgrpc

import (
	"context"
	"encoding/json"

	"slices"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	HealthMethod = "/grpc.health.v1.Health/Check"
)

// 请求参数拦截器
// ignoreMethods: 忽略的方法, 例如: []string{"/grpc.health.v1.Health/Check"}
func RequestParamInterceptor(enableLog bool, ignoreMethods []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 忽略ignoreMethods中的方法
		if slices.Contains(ignoreMethods, info.FullMethod) {
			return handler(ctx, req)
		}

		// 打印请求参数
		reqJSON, err := json.Marshal(req)
		if err != nil {
			Log(enableLog, zerolog.ErrorLevel, "Failed to marshal request: %v", err)
		} else {
			Log(enableLog, zerolog.InfoLevel, "gRPC call incoming - method: %s, req: %s",
				info.FullMethod, string(reqJSON))
		}

		// 处理请求
		resp, err := handler(ctx, req)
		return resp, err
	}
}

// 响应拦截器
// ignoreMethods: 忽略的方法, 例如: []string{"/grpc.health.v1.Health/Check"}
// maxRespSize: 最大返回值长度
func ResponseInterceptor(enableLog bool, ignoreMethods []string, maxRespSize int) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 忽略ignoreMethods中的方法
		if slices.Contains(ignoreMethods, info.FullMethod) {
			return handler(ctx, req)
		}

		// 处理请求
		resp, err := handler(ctx, req)
		if err != nil {
			Log(enableLog, zerolog.ErrorLevel, "gRPC call failed - method: %s, error: %v",
				info.FullMethod, err)
		} else {
			// 打印返回值
			respJSON, err := json.Marshal(resp)
			if err != nil {
				Log(enableLog, zerolog.ErrorLevel, "Failed to marshal response: %v", err)
			} else {
				if len(respJSON) > maxRespSize {
					respJSON = append(respJSON[:maxRespSize], []byte("...")...)
					Log(enableLog, zerolog.InfoLevel, "gRPC call completed - method: %s, resp: %s",
						info.FullMethod, string(respJSON))
				} else {
					Log(enableLog, zerolog.InfoLevel, "gRPC call completed - method: %s, resp: %s",
						info.FullMethod, string(respJSON))
				}
			}
		}
		return resp, err
	}
}

func ValidatorOption(enableLog bool) grpc.ServerOption {
	return grpc.UnaryInterceptor(ValidatorInterceptor(enableLog))
}

// 定义一个接口，用于请求验证
type GrpcValidator interface {
	Validate() error
}

// 请求验证中间件
func ValidatorInterceptor(enableLog bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if v, ok := req.(GrpcValidator); ok {
			if err := v.Validate(); err != nil {
				Log(enableLog, zerolog.ErrorLevel, "gRpc request validation failed: %v", err)
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		return handler(ctx, req)
	}
}
