package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sunliang711/ez-go/ezhttp/middleware"
	"github.com/sunliang711/ez-go/ezhttp/server"
	"github.com/sunliang711/ez-go/ezhttp/utils"
	"github.com/sunliang711/ez-go/ezlog"
)

var secret = "a-string-secret-at-least-256-bits-long"

func main() {
	ezlog.SetLog(ezlog.WithTimestamp(true), ezlog.WithWriter(ezlog.Stdout), ezlog.WithLevel(zerolog.TraceLevel), ezlog.WithServiceName("ezhttp_example"))

	httpServer := server.NewHttpServer(server.WithPort(9001), server.EnableLog(true))
	httpServer.AddHealthHandler()

	// global middlewares
	// httpServer.AddMiddlewares([]server.Middleware{
	// 	{
	// 		Name:    "jwt-checker",
	// 		Handler: middleware.JwtChecker("a"),
	// 	},
	// })
	assetManager := &AssetManager{}
	UserManager := &UserManager{}
	err := httpServer.AddRoutes([]server.Routes{
		{
			GroupPath:        "/user",
			GroupMiddlewares: []gin.HandlerFunc{},
			Handlers: []server.Handler{
				{
					Name:        "login",
					Method:      "POST",
					Path:        "/login",
					Middlewares: []gin.HandlerFunc{},
					Handler:     UserManager.Login,
				},
				{
					Name:   "userInfo",
					Method: "GET",
					Path:   "/info",
					Middlewares: []gin.HandlerFunc{

						middleware.JwtChecker(true, secret, middleware.JwtInfo2GinContext{
							ContextKey: "username",
							JwtKey:     "username",
						}),
					},
					Handler: UserManager.UserInfo,
				},
			},
		},
		{
			GroupPath: "/blockchain",
			GroupMiddlewares: []gin.HandlerFunc{
				middleware.JwtChecker(true, "a-string-secret-at-least-256-bits-long", middleware.JwtInfo2GinContext{
					ContextKey: "username",
					JwtKey:     "username",
				}),
			},
			Handlers: []server.Handler{
				{
					Name:        "blockchain",
					Method:      "GET",
					Path:        "/list",
					Middlewares: []gin.HandlerFunc{},
					Handler:     ListBlockchain,
				},
				{
					Name:        "listAsset",
					Method:      "GET",
					Path:        "/listAsset",
					Middlewares: []gin.HandlerFunc{},
					Handler:     assetManager.ListAssets,
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	httpServer.AddCustomFunc(func(e *gin.Engine) {
		v2 := e.Group("/api/v2")
		v2.GET("/health", func(c *gin.Context) {
			c.JSON(200, "ok_v2")
		})
	})

	httpServer.Start()

	// wait signal
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, os.Interrupt, syscall.SIGTERM)
	<-osSignal

	httpServer.Stop()
}

func ListBlockchain(c *gin.Context) {
	blockchains := []Blockchain{
		{
			BlockchainName: "Ethereum",
			ID:             0,
		},
		{
			BlockchainName: "Bitcoin",
			ID:             1,
		},
	}
	if name, ok := c.Get("name"); ok {
		log.Info().Msgf("name from context: %v", name)
	}

	c.JSON(0, blockchains)
}

type Blockchain struct {
	BlockchainName string `json:"blockchain_name"`
	ID             int    `json:"id"`
}

type AssetManager struct{}

func (a *AssetManager) ListAssets(c *gin.Context) {
	c.JSON(0, []string{"BTC", "ETH"})
}

type UserManager struct{}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// login
func (u *UserManager) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Here you would typically validate the user credentials and generate a JWT token.
	// For simplicity, we are returning a mock token.
	jwtToken, err := utils.GenerateJwtToken(secret, 30, map[string]any{
		"username": req.Username,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate token"})
		return
	}

	if req.Username != "admin" || req.Password != "password" {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(200, gin.H{"token": jwtToken, "message": "login successful"})
}

// user info
func (u *UserManager) UserInfo(c *gin.Context) {
	// get username from context
	username, exists := c.Get("username")
	if !exists {
		c.JSON(400, gin.H{"error": "username not found in context"})
		return
	}
	c.JSON(200, gin.H{
		"username": username,
		"message":  "user info retrieved successfully",
	})
}
