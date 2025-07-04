package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/sunliang711/ez-go/ezhttp/utils"

	swagFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type HttpServer struct {
	gin *gin.Engine

	server            *http.Server
	globalMiddlewares []Middleware

	enableSwag bool

	enableCors bool
	corsConfig cors.Config
	// jwtSecret  string

	// logger *log.Logger
	//
	enableLog bool

	routes []Routes

	customFuncs []CustomFunc
}

type serverOptions struct {
	host       string
	port       int
	enableSwag bool
	enableCors bool
	corsConfig cors.Config
	enableLog  bool
}
type ServerOption func(*serverOptions)

func WithHost(host string) ServerOption {
	return func(o *serverOptions) {
		o.host = host
	}
}

func WithPort(port int) ServerOption {
	return func(o *serverOptions) {
		o.port = port
	}
}

func WithSwag(enableSwag bool) ServerOption {
	return func(o *serverOptions) {
		o.enableSwag = enableSwag
	}
}

func WithCors(enableCors bool) ServerOption {
	return func(o *serverOptions) {
		o.enableCors = enableCors
	}
}

func WithCorsConfig(corsConfig cors.Config) ServerOption {
	return func(o *serverOptions) {
		o.corsConfig = corsConfig
	}
}

// EnableLog enables logging
func EnableLog() ServerOption {
	return func(o *serverOptions) {
		o.enableLog = true
	}
}

// func NewHttpServer(host string, port int, enableSwag, enableCors bool, corsConfig cors.Config) *HttpServer {
func NewHttpServer(options ...ServerOption) *HttpServer {
	defaultOptions := &serverOptions{
		host:       "0.0.0.0",
		port:       9000,
		enableSwag: false,
		enableCors: false,
		corsConfig: cors.Config{},
		enableLog:  false,
	}

	for _, opt := range options {
		opt(defaultOptions)
	}

	ginEngine := gin.New()
	ginEngine.Use(gin.Logger(), gin.Recovery())

	if defaultOptions.host == "" {
		defaultOptions.host = "0.0.0.0"
	}

	addr := fmt.Sprintf("%s:%d", defaultOptions.host, defaultOptions.port)
	srv := &http.Server{
		Addr:    addr,
		Handler: ginEngine,
	}

	return &HttpServer{
		server: srv,
		gin:    ginEngine,
		// logger:     log.New(os.Stdout, "|HTTP_SERVER| ", log.LstdFlags),
		enableSwag: defaultOptions.enableSwag,
		enableCors: defaultOptions.enableCors,
		corsConfig: defaultOptions.corsConfig,
		enableLog:  defaultOptions.enableLog,
		// jwtSecret:  jwtSecret,
	}
}

func (s *HttpServer) setupSwag() {
	if !s.enableSwag {
		return
	}
	// setup swag
	utils.Log(s.enableLog, zerolog.InfoLevel, "setup swag")
	s.gin.GET("/swagger/*any", ginSwagger.WrapHandler(swagFiles.Handler))
}

func (s *HttpServer) setupCors() {
	if !s.enableCors {
		return
	}
	// setup cors
	utils.Log(s.enableLog, zerolog.InfoLevel, "setup cors ")
	s.AddGlobalMiddlewares([]Middleware{{Name: "gin-cors", Handler: cors.New(s.corsConfig)}})

}

func (s *HttpServer) GetEngine() *gin.Engine {
	return s.gin
}

func (s *HttpServer) AddGlobalMiddlewares(mws []Middleware) {
	s.globalMiddlewares = append(s.globalMiddlewares, mws...)
}

func (s *HttpServer) setupGlobalMiddlewares() {
	for _, middleware := range s.globalMiddlewares {
		utils.Log(s.enableLog, zerolog.InfoLevel, "setup global middleware: %s", middleware.Name)
		s.gin.Use(middleware.Handler)
	}
}

func (s *HttpServer) AddRoutes(routes []Routes) error {
	// check handlers
	for _, r := range routes {
		for _, h := range r.Handlers {
			if h.Method == "" {
				return fmt.Errorf("method is empty")
			}
			if h.Handler == nil {
				return fmt.Errorf("handler is nil")
			}
			if h.Path == "" {
				return fmt.Errorf("path is empty")
			}
		}
	}
	s.routes = append(s.routes, routes...)

	return nil
}

func (s *HttpServer) setupRoutes() {

	for _, routes := range s.routes {
		group := s.gin.Group(routes.GroupPath)
		if len(routes.GroupMiddlewares) > 0 {
			group.Use(routes.GroupMiddlewares...)
		}

		for _, handler := range routes.Handlers {
			// get middlewaresAndHandler
			middlewaresAndHandler := []gin.HandlerFunc{}
			// 添加中间件
			middlewaresAndHandler = append(middlewaresAndHandler, handler.Middlewares...)
			// 添加handler
			middlewaresAndHandler = append(middlewaresAndHandler, handler.Handler)

			utils.Log(s.enableLog, zerolog.InfoLevel, "setup route: %s %s", handler.Method, routes.GroupPath+handler.Path)
			switch handler.Method {
			case http.MethodPost:
				group.POST(handler.Path, middlewaresAndHandler...)
			case http.MethodGet:
				group.GET(handler.Path, middlewaresAndHandler...)
			case http.MethodPut:
				group.PUT(handler.Path, middlewaresAndHandler...)
			case http.MethodDelete:
				group.DELETE(handler.Path, middlewaresAndHandler...)
			default:
				utils.Log(s.enableLog, zerolog.ErrorLevel, "setup routes: unsupport HTTP method: %s", handler.Method)
			}
		}
	}
}

func (s *HttpServer) start() {
	utils.Log(s.enableLog, zerolog.InfoLevel, "start http server on: %s", s.server.Addr)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("listen: %s\n", err))
		}
	}()
}

// add health handler
func (s *HttpServer) AddHealthHandler() {
	s.AddRoutes([]Routes{{
		GroupPath: "/api/v1",
		Handlers: []Handler{
			{
				Name:   "health",
				Method: "GET",
				Path:   "/health",
				Handler: func(c *gin.Context) {
					c.String(http.StatusOK, "ok")
				},
			},
		},
	}})
}

func (s *HttpServer) AddCustomFunc(f CustomFunc) {
	s.customFuncs = append(s.customFuncs, f)
}

func (s *HttpServer) executeCustomFunc() {
	for _, f := range s.customFuncs {
		f(s.gin)
	}
}

func (s *HttpServer) Start() error {
	// 设置跨域
	s.setupCors()

	// 设置中间件
	s.setupGlobalMiddlewares()

	// 设置swagger
	s.setupSwag()

	// 设置路由
	s.setupRoutes()

	// 自定义函数
	s.executeCustomFunc()

	// 启动服务
	s.start()

	return nil
}

func (s *HttpServer) Stop() error {
	utils.Log(s.enableLog, zerolog.InfoLevel, "shutdown http server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优雅关闭服务
	err := s.server.Shutdown(ctx)
	if err != nil {
		utils.Log(s.enableLog, zerolog.ErrorLevel, "shutdown http server error: %v", err)
		return err
	}

	return nil
}
