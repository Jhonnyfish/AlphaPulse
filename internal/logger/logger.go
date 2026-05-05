package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 根据环境变量切换日志级别
	if os.Getenv("APP_ENV") == "development" {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
	}

	var err error
	log, err = config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
}

// L returns the global logger instance.
func L() *zap.Logger {
	return log
}

// Sync flushes any buffered log entries.
func Sync() {
	log.Sync()
}

// WithFields creates a child logger with fields.
func WithFields(fields ...zap.Field) *zap.Logger {
	return log.With(fields...)
}

// Info logs a message at info level.
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Error logs a message at error level.
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

// Debug logs a message at debug level.
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Warn logs a message at warn level.
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// Fatal logs a message at fatal level then calls os.Exit(1).
func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

// WithError creates a logger with an error field.
func WithError(err error) *zap.Logger {
	return log.With(zap.Error(err))
}

// WithRequestID creates a logger with a request ID field.
func WithRequestID(id string) *zap.Logger {
	return log.With(zap.String("request_id", id))
}
