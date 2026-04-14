package logging

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"backend/internal/config"
)

var logger *zap.Logger

var sensitiveFields = []string{"password", "token", "ssn", "secret", "authorization", "cookie"}

func InitLogger(cfg *config.Config) {
	level := parseLogLevel(cfg.LogLevel)

	zapCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      cfg.AppEnv != "production",
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	var err error
	logger, err = zapCfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
}

func GetLogger() *zap.Logger {
	if logger == nil {
		fallback, _ := zap.NewProduction()
		return fallback
	}
	return logger
}

func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

func Info(module, action, message string, fields ...zap.Field) {
	GetLogger().Info(
		formatMessage(module, action, message),
		fields...,
	)
}

func Warn(module, action, message string, fields ...zap.Field) {
	GetLogger().Warn(
		formatMessage(module, action, message),
		fields...,
	)
}

func Error(module, action, message string, fields ...zap.Field) {
	GetLogger().Error(
		formatMessage(module, action, message),
		fields...,
	)
}

func Debug(module, action, message string, fields ...zap.Field) {
	GetLogger().Debug(
		formatMessage(module, action, message),
		fields...,
	)
}

func formatMessage(module, action, message string) string {
	return fmt.Sprintf("[%s][%s] %s", module, action, message)
}

func RedactValue(key, value string) string {
	lowerKey := strings.ToLower(key)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerKey, sensitive) {
			if len(value) <= 4 {
				return "****"
			}
			return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
		}
	}
	return value
}

func IsSensitiveField(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerKey, sensitive) {
			return true
		}
	}
	return false
}

func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		correlationID, _ := c.Get("correlation_id")

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", RedactQueryString(query)),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.Int("body_size", c.Writer.Size()),
		}

		if cid, ok := correlationID.(string); ok {
			fields = append(fields, zap.String("correlation_id", cid))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		if status >= 500 {
			Error("http", "request", "server error", fields...)
		} else if status >= 400 {
			Warn("http", "request", "client error", fields...)
		} else {
			Info("http", "request", "completed", fields...)
		}
	}
}

func RedactQueryString(query string) string {
	if query == "" {
		return ""
	}
	pairs := strings.Split(query, "&")
	redacted := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 && IsSensitiveField(parts[0]) {
			redacted = append(redacted, parts[0]+"=****")
		} else {
			redacted = append(redacted, pair)
		}
	}
	return strings.Join(redacted, "&")
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
