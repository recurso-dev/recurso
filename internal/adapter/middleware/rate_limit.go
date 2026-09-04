package middleware

import (
	"fmt"
	"log/slog"
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
//
// When Redis is unavailable (nil client, or a transient error) the limiter
// falls back to a per-process in-memory window rather than admitting the
// request unchecked: this middleware is what bounds login, forgot-password and
// SAML brute force, so a Redis blip must not switch that off. The fallback is
// per instance (so N replicas allow N× the limit) — a degraded ceiling, not
// none. Expired in-memory entries are pruned once per window so a long outage
// does not grow the map without bound.
func RateLimitMiddleware(rdb *redis.Client, scope string, limit int, window time.Duration) gin.HandlerFunc {
	var (
		mu        sync.Mutex
		counters  = make(map[string]*rateLimitEntry)
		lastSweep = time.Now()
		warnedAt  time.Time
	)

	local := func(key string) int64 {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if now.Sub(lastSweep) > window {
			for k, e := range counters {
				if now.After(e.expiresAt) {
					delete(counters, k)
				}
			}
			lastSweep = now
		}
		entry, exists := counters[key]
		if !exists || now.After(entry.expiresAt) {
			entry = &rateLimitEntry{count: 0, expiresAt: now.Add(window)}
			counters[key] = entry
		}
		entry.count++
		return entry.count
	}

	return func(c *gin.Context) {
		// Key based on IP or Tenant if available
		key := fmt.Sprintf("ratelimit:%s:%s", scope, c.ClientIP())
		if tenantID, exists := c.Get("tenant_id"); exists {
			key = fmt.Sprintf("ratelimit:%s:tenant:%v", scope, tenantID)
		}

		var count int64
		if rdb != nil {
			n, err := rdb.Incr(c.Request.Context(), key).Result()
			if err != nil {
				mu.Lock()
				if time.Since(warnedAt) > window {
					warnedAt = time.Now()
					slog.Warn("rate limiter: redis unavailable, using in-memory window", "scope", scope, "error", err)
				}
				mu.Unlock()
				count = local(key)
			} else {
				if n == 1 {
					rdb.Expire(c.Request.Context(), key, window)
				}
				count = n
			}
		} else {
			count = local(key)
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
