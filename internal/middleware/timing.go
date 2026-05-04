package middleware

import (
	"time"

	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TimingMiddleware records per-request performance metrics via PerfTracker.
// Place it after RequestLogger in the middleware chain so both fire.
func TimingMiddleware(tracker *services.PerfTracker) gin.HandlerFunc {
	log := zap.L()

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start)
		durationMs := float64(duration) / float64(time.Millisecond)

		tracker.RecordRequest(
			c.Request.Method,
			c.Request.URL.Path,
			durationMs,
			c.Writer.Status(),
			c.ClientIP(),
		)

		if duration > services.SlowRequestThreshold {
			log.Warn("slow request detected",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Float64("duration_ms", durationMs),
				zap.Int("status", c.Writer.Status()),
				zap.String("ip", c.ClientIP()),
			)
		}
	}
}
