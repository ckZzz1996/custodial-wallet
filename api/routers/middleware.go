package routers

import (
	"strings"
	"sync"
	"time"

	"custodial-wallet/pkg/httputil"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

// SetJWTSecret 设置JWT密钥
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httputil.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			httputil.Unauthorized(c, "invalid authorization header")
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			httputil.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httputil.Unauthorized(c, "invalid token claims")
			c.Abort()
			return
		}

		userID := uint(claims["user_id"].(float64))
		c.Set("user_id", userID)
		c.Set("user_uuid", claims["uuid"])
		c.Set("user_email", claims["email"])

		c.Next()
	}
}

// APIKeyMiddleware API密钥认证中间件（占位，需注入account service for real validation）
func APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		apiSecret := c.GetHeader("X-API-Secret")

		if apiKey == "" || apiSecret == "" {
			httputil.Unauthorized(c, "missing API credentials")
			c.Abort()
			return
		}

		// NOTE: real validation requires account service; currently allow through
		c.Next()
	}
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-API-Key, X-API-Secret")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware 简单的内存限流器（每IP每秒允许请求数）
func RateLimitMiddleware() gin.HandlerFunc {
	var mu sync.Mutex
	visitors := make(map[string]*visitor)
	go cleanupVisitors(&mu, visitors)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		v, ok := visitors[ip]
		if !ok {
			v = &visitor{last: time.Now(), tokens: 10}
			visitors[ip] = v
		}
		// refill tokens
		now := time.Now()
		delta := now.Sub(v.last).Seconds()
		v.tokens += int(delta) // 1 token per second
		if v.tokens > 100 {
			v.tokens = 100
		}
		v.last = now
		if v.tokens <= 0 {
			mu.Unlock()
			httputil.Error(c, 429, "too many requests")
			c.Abort()
			return
		}
		v.tokens--
		mu.Unlock()
		c.Next()
	}
}

type visitor struct {
	last   time.Time
	tokens int
}

func cleanupVisitors(mu *sync.Mutex, visitors map[string]*visitor) {
	for {
		time.Sleep(1 * time.Minute)
		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.last) > 10*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return gin.Logger()
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}
