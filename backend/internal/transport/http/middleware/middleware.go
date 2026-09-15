// Package middleware: request id, logging, recovery, CORS, auth, rate limiting, body limits.
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/httpx"
	"yingji/backend/internal/pkg/idgen"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" || len(id) > 64 {
			id = idgen.New()
		}
		c.Set(httpx.CtxRequestID, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if strings.HasPrefix(c.Request.URL.Path, "/healthz") || strings.HasPrefix(c.Request.URL.Path, "/metrics") {
			return
		}
		log.Info("http", "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(),
			"ms", time.Since(start).Milliseconds(), "user_id", httpx.UserID(c), "request_id", httpx.RequestID(c))
	}
}

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic", "err", r, "path", c.Request.URL.Path, "request_id", httpx.RequestID(c))
				httpx.Fail(c, apperr.Internal(fmt.Errorf("panic: %v", r)))
			}
		}()
		c.Next()
	}
}

func CORS(origins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (allowed[origin] || allowed["*"]) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id, X-Client-Version, X-Platform, Idempotency-Key")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// UserAuth validates the Bearer token via parse and stores the user id.
func UserAuth(parse func(token string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := bearer(c)
		if tok == "" {
			httpx.Fail(c, apperr.Unauthorized())
			return
		}
		uid, err := parse(tok)
		if err != nil {
			httpx.Fail(c, apperr.Unauthorized())
			return
		}
		c.Set(httpx.CtxUserID, uid)
		c.Next()
	}
}

// OptionalUser sets the user id when a valid token is present, without requiring one.
func OptionalUser(parse func(token string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tok := bearer(c); tok != "" {
			if uid, err := parse(tok); err == nil {
				c.Set(httpx.CtxUserID, uid)
			}
		}
		c.Next()
	}
}

func AdminAuth(parse func(token string) (id, role string, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := bearer(c)
		id, role, err := parse(tok)
		if tok == "" || err != nil {
			httpx.Fail(c, apperr.Unauthorized())
			return
		}
		c.Set(httpx.CtxAdminID, id)
		c.Set(httpx.CtxAdminRole, role)
		c.Next()
	}
}

func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// RateLimit is a fixed-window limiter in Redis keyed by user (or IP) and name.
func RateLimit(rdb *redis.Client, name string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}
		who := httpx.UserID(c)
		if who == "" {
			who = c.ClientIP()
		}
		key := fmt.Sprintf("rl:%s:%s:%d", name, who, time.Now().Unix()/int64(window.Seconds()))
		n, err := rdb.Incr(c, key).Result()
		if err == nil && n == 1 {
			rdb.Expire(c, key, window)
		}
		if err == nil && n > int64(limit) {
			httpx.Fail(c, apperr.RateLimited())
			return
		}
		c.Next()
	}
}

func BodyLimit(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		c.Next()
	}
}
