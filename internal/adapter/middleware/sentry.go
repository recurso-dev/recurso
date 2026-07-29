package middleware

import (
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// SentryMiddleware reports panics and 5xx handler errors to Sentry. It is inert
// unless Sentry was initialized (SENTRY_DSN set) — sentry calls are no-ops
// without a configured client. It sits INSIDE gin.Default()'s Recovery, so a
// panic is captured here and then re-raised for gin to turn into a 500.
func SentryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("path", c.FullPath())
		hub.Scope().SetTag("method", c.Request.Method)

		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(c.Request.Context(), err)
				hub.Flush(2 * time.Second)
				panic(err) // let gin.Recovery return the 500
			}
		}()

		c.Next()

		// Report handler-recorded errors that resulted in a 5xx.
		if c.Writer.Status() >= 500 {
			for _, e := range c.Errors {
				hub.CaptureException(e.Err)
			}
		}
	}
}
