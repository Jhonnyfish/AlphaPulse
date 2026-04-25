package middleware

import (
	"time"

	"alphapulse/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", "*")
		headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		headers.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}

		switch {
		case statusCode >= 500:
			logger.Error("Server Error", fields...)
		case statusCode >= 400:
			logger.Warn("Client Error", fields...)
		default:
			logger.Info("Request", fields...)
		}
	}
}
