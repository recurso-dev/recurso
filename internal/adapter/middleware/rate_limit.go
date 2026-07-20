package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/recurso-dev/recurso/internal/adapter/httperr"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware implements a fixed-window rate limiter using Redis.
//
// scope namespaces the counter key. Limiters with different limits MUST use
// different scopes: with a shared key, every request drains one bucket, and
// the strictest limiter judges the combined total — the global 500/min
// middleware plus the 20/min auth limiter once shared "ratelimit:<ip>", so
// ~20 requests of ANY kind per minute made /auth/me and /auth/login return
// 429 (surfacing in the dashboard as login bounces and "Could not reach the
// API" on the login screen).
func RateLimitMiddleware(rdb *redis.Client, scope string, limit int, window time.Duration) gin.HandlerFunc {
	// In-memory fallback when Redis is not available
	var mu sync.Mutex
	counters := make(map[string]*rateLimitEntry)

	return func(c *gin.Context) {
		// Key based on IP or Tenant if available
		key := fmt.Sprintf("ratelimit:%s:%s", scope, c.ClientIP())
		if tenantID, exists := c.Get("tenant_id"); exists {
			key = fmt.Sprintf("ratelimit:%s:tenant:%v", scope, tenantID)
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
