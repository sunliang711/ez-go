package ezgrpc

import (
	"context"
	"encoding/json"

	"slices"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	HealthMethod = "/grpc.health.v1.Health/Check"
)

// 请求参数拦截器
// ignoreMethods: 忽略的方法, 例如: []string{"/grpc.health.v1.Health/Check"}
func RequestParamInterceptor(ignoreMethods []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		log := log.With().Str("mw", "GrpcReq").Logger()
		// 忽略ignoreMethods中的方法
		if slices.Contains(ignoreMethods, info.FullMethod) {
			return handler(ctx, req)
		}

		// 打印请求参数
		reqJSON, err := json.Marshal(req)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal request")
		} else {
			log.Info().
				Str("method", info.FullMethod).
				RawJSON("req", reqJSON).
				Msg("gRPC call incoming")
		}

		// 处理请求
		resp, err := handler(ctx, req)
		return resp, err
	}
}

// 响应拦截器
// ignoreMethods: 忽略的方法, 例如: []string{"/grpc.health.v1.Health/Check"}
// maxRespSize: 最大返回值长度
func ResponseInterceptor(ignoreMethods []string, maxRespSize int) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		log := log.With().Str("mw", "GrpcResp").Logger()
		// 忽略ignoreMethods中的方法
		if slices.Contains(ignoreMethods, info.FullMethod) {
			return handler(ctx, req)
		}

		// 处理请求
		resp, err := handler(ctx, req)
		if err != nil {
			log.Error().Err(err).
				Str("method", info.FullMethod).
				Msg("gRPC call failed")
		} else {
			// 打印返回值
			respJSON, err := json.Marshal(resp)
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal response")
			} else {
				if len(respJSON) > maxRespSize {
					respJSON = append(respJSON[:maxRespSize], []byte("...")...)
					log.Info().
						Str("method", info.FullMethod).
						Str("resp", string(respJSON)).
						Msg("gRPC call completed")
				} else {
					log.Info().
						Str("method", info.FullMethod).
						RawJSON("res", respJSON).
						Msg("gRPC call completed")
				}
			}
		}
		return resp, err
	}
}

func ValidatorOption() grpc.ServerOption {
	return grpc.UnaryInterceptor(ValidatorInterceptor())
}

// 定义一个接口，用于请求验证
type GrpcValidator interface {
	Validate() error
}

// 请求验证中间件
func ValidatorInterceptor() grpc.UnaryServerInterceptor {
	log := log.With().Str("fn", "ValidatorInterceptor").Logger()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if v, ok := req.(GrpcValidator); ok {
			if err := v.Validate(); err != nil {
				log.Error().Err(err).Msg("gRpc request validation failed")
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		return handler(ctx, req)
	}
}
