package middleware

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/sunliang711/ez-go/ezhttp/types"
	"github.com/sunliang711/ez-go/ezhttp/utils"
)

const (
	jwtHeaderName = "Authorization"
)

// JwtInfo2GinContext is a struct that holds the context key and JWT key-value pair.
// 把JWT信息存储到gin.Context中，方便后续使用
type JwtInfo2GinContext struct {
	ContextKey string
	JwtKey     string
	// JwtValue   string
}

// JwtChecker is a middleware function that checks the JWT token in the request header.
// It parses the token and stores the specified JWT claims in the Gin context.
func JwtChecker(enableLog bool, secret string, jwtInfo2GinContext ...JwtInfo2GinContext) func(c *gin.Context) {
	return func(c *gin.Context) {

		// requestId, _ := c.Get(consts.ContextKeyRequestId)

		// Get header
		token := c.Request.Header.Get(jwtHeaderName)
		if token == "" {
			utils.Log(enableLog, zerolog.ErrorLevel, "cannot get jwt header %s", jwtHeaderName)
			c.AbortWithStatusJSON(http.StatusBadRequest, types.Response{
				// RequestId: requestId.(string),
				// Success: false,
				Code: types.CodeGeneralError,
				Msg:  "need jwt token",
				Data: nil,
			})
			return
		}

		// Parse token
		parsedToken, err := utils.ParseJwtToken(token, secret)
		if err != nil {
			// logger.Error().Err(err).Msg("parse jwt token")
			utils.Log(enableLog, zerolog.ErrorLevel, "parse jwt token error: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, types.Response{
				// RequestId: requestId.(string),
				// Success:   false,
				Code: types.CodeGeneralError,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}

		// Type assertion
		claims, OK := parsedToken.Claims.(jwt.MapClaims)
		if !OK {
			// logger.Error().Msg("parsed token type assertion failed")
			utils.Log(enableLog, zerolog.ErrorLevel, "parsed token type assertion failed")
			c.AbortWithStatusJSON(http.StatusInternalServerError, types.Response{
				// RequestId: requestId.(string),
				// Success:   false,
				Code: types.CodeGeneralError,
				Msg:  "unable to parse claims",
				Data: nil,
			})
			return
		}

		// Store JWT claims in context
		for _, jwtInfo := range jwtInfo2GinContext {
			if value, exists := claims[jwtInfo.JwtKey]; exists {
				utils.Log(enableLog, zerolog.DebugLevel, "set jwt key: %s value: %v to gin context key: %s", jwtInfo.JwtKey, value, jwtInfo.ContextKey)
				c.Set(jwtInfo.ContextKey, value)
			} else {
				utils.Log(enableLog, zerolog.WarnLevel, "jwt key %s not found in claims", jwtInfo.JwtKey)
				continue
			}
		}

		c.Next()
	}

}
