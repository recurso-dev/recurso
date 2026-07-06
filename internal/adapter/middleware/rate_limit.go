package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/swapnull-in/recur-so/internal/adapter/httperr"
)

// RateLimitMiddleware implements a fixed-window rate limiter using Redis
func RateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	// In-memory fallback when Redis is not available
	var mu sync.Mutex
	counters := make(map[string]*rateLimitEntry)

	return func(c *gin.Context) {
		// Key based on IP or Tenant if available
		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())
		if tenantID, exists := c.Get("tenant_id"); exists {
			key = fmt.Sprintf("ratelimit:tenant:%v", tenantID)
		}

		var count int64

		if rdb != nil {
			var err error
			count, err = rdb.Incr(c.Request.Context(), key).Result()
			if err != nil {
				c.Next()
				return
			}
			if count == 1 {
				rdb.Expire(c.Request.Context(), key, window)
			}
		} else {
			// In-memory rate limiting
			mu.Lock()
			entry, exists := counters[key]
			now := time.Now()
			if !exists || now.After(entry.expiresAt) {
				entry = &rateLimitEntry{count: 0, expiresAt: now.Add(window)}
				counters[key] = entry
			}
			entry.count++
			count = entry.count
			mu.Unlock()
		}

		if count > int64(limit) {
			httperr.Abort(c, http.StatusTooManyRequests, httperr.CodeRateLimited, "Too many requests")
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", int64(limit)-count))
		c.Next()
	}
}

type rateLimitEntry struct {
	count     int64
	expiresAt time.Time
}
