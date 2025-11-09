package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/rajweepmondal/url-shortener/internal/config"
)

// NewLogger creates a new logger instance based on configuration
func NewLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	// Choose base configuration
	if cfg.Format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Set log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Add caller information for development
	if cfg.Format == "console" {
		zapConfig.Development = true
		zapConfig.EncoderConfig.CallerKey = "caller"
		zapConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	// Build logger
	logger, err := zapConfig.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewStructuredLogger creates a structured logger with additional fields
func NewStructuredLogger(cfg config.LogConfig, service string, version string) (*zap.Logger, error) {
	logger, err := NewLogger(cfg)
	if err != nil {
		return nil, err
	}

	// Add service-level fields
	return logger.With(
		zap.String("service", service),
		zap.String("version", version),
		zap.Int("pid", os.Getpid()),
	), nil
}

// LoggerMiddleware provides logging utilities for middleware
type LoggerMiddleware struct {
	logger *zap.Logger
}

// NewLoggerMiddleware creates a new logger middleware
func NewLoggerMiddleware(logger *zap.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{logger: logger}
}

// WithFields adds fields to the logger
func (lm *LoggerMiddleware) WithFields(fields ...zap.Field) *zap.Logger {
	return lm.logger.With(fields...)
}

// LogRequest logs an incoming request
func (lm *LoggerMiddleware) LogRequest(method, path string, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
	}
	baseFields = append(baseFields, fields...)
	lm.logger.Info("incoming request", baseFields...)
}

// LogResponse logs a response
func (lm *LoggerMiddleware) LogResponse(method, path string, statusCode int, duration int64, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
		zap.Int64("duration_ms", duration),
	}
	baseFields = append(baseFields, fields...)

	if statusCode >= 400 {
		lm.logger.Error("request failed", baseFields...)
	} else {
		lm.logger.Info("request completed", baseFields...)
	}
}

// LogError logs an error with context
func (lm *LoggerMiddleware) LogError(err error, message string, fields ...zap.Field) {
	baseFields := []zap.Field{zap.Error(err)}
	baseFields = append(baseFields, fields...)
	lm.logger.Error(message, baseFields...)
}

// LogPanic logs a panic with stack trace
func (lm *LoggerMiddleware) LogPanic(recovered interface{}, stack []byte, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.Any("panic", recovered),
		zap.ByteString("stack", stack),
	}
	baseFields = append(baseFields, fields...)
	lm.logger.Error("panic recovered", baseFields...)
}

// ContextLogger provides context-aware logging
type ContextLogger struct {
	logger *zap.Logger
	fields []zap.Field
}

// NewContextLogger creates a new context logger
func NewContextLogger(logger *zap.Logger) *ContextLogger {
	return &ContextLogger{logger: logger}
}

// WithField adds a field to the context logger
func (cl *ContextLogger) WithField(key string, value interface{}) *ContextLogger {
	return &ContextLogger{
		logger: cl.logger,
		fields: append(cl.fields, zap.Any(key, value)),
	}
}

// WithFields adds multiple fields to the context logger
func (cl *ContextLogger) WithFields(fields ...zap.Field) *ContextLogger {
	return &ContextLogger{
		logger: cl.logger,
		fields: append(cl.fields, fields...),
	}
}

// Debug logs a debug message
func (cl *ContextLogger) Debug(msg string, fields ...zap.Field) {
	allFields := append(cl.fields, fields...)
	cl.logger.Debug(msg, allFields...)
}

// Info logs an info message
func (cl *ContextLogger) Info(msg string, fields ...zap.Field) {
	allFields := append(cl.fields, fields...)
	cl.logger.Info(msg, allFields...)
}

// Warn logs a warning message
func (cl *ContextLogger) Warn(msg string, fields ...zap.Field) {
	allFields := append(cl.fields, fields...)
	cl.logger.Warn(msg, allFields...)
}

// Error logs an error message
func (cl *ContextLogger) Error(msg string, fields ...zap.Field) {
	allFields := append(cl.fields, fields...)
	cl.logger.Error(msg, allFields...)
}

// Fatal logs a fatal message and exits
func (cl *ContextLogger) Fatal(msg string, fields ...zap.Field) {
	allFields := append(cl.fields, fields...)
	cl.logger.Fatal(msg, allFields...)
}

// Sync flushes any buffered log entries
func (cl *ContextLogger) Sync() error {
	return cl.logger.Sync()
}

// LogLevel represents log levels
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
)

// IsValidLogLevel checks if a log level is valid
func IsValidLogLevel(level string) bool {
	switch LogLevel(level) {
	case DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel:
		return true
	default:
		return false
	}
}

// GetDefaultLogger returns a default logger for quick use
func GetDefaultLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}
