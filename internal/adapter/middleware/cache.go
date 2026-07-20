package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// CacheMiddleware caches GET responses in Redis
func CacheMiddleware(rdb *redis.Client, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil || c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Tenant-scoped key only. Without a resolved tenant there is no safe
		// cache identity — the old ClientIP fallback could serve one user's
		// payload to another behind a shared NAT/proxy if this middleware were
		// ever mounted before auth. Skip caching instead.
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.Next()
			return
		}
		key := fmt.Sprintf("cache:%s:%v", c.Request.URL.String(), tenantID)

		// Check Cache
		val, err := rdb.Get(c.Request.Context(), key).Result()
		if err == nil {
			c.Header("X-Cache", "HIT")
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.String(http.StatusOK, val)
			c.Abort()
			return
		}

		// Cache Miss - Capture Response
		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w
		c.Header("X-Cache", "MISS")
		c.Next()

		// Store in Cache if 200 OK
		if c.Writer.Status() == http.StatusOK {
			rdb.Set(c.Request.Context(), key, w.body.String(), ttl)
		}
	}
}
