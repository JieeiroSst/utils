package logger

import (
	"context"
	"os"

	"github.com/JIeeiroSst/utils/trace_id"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	zap *zap.Logger
}

type Config struct {
	Level      string
	JSONFormat bool
	AppName    string
}

func New(config Config) (*Logger, error) {
	level := zap.InfoLevel
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zap.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if config.JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)

	zapLogger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.Fields(zap.String("app", config.AppName)),
	)

	return &Logger{zap: zapLogger}, nil
}

func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	if tracerID := trace_id.GetTracerID(ctx); tracerID != "" {
		return &Logger{zap: l.zap.With(zap.String("trace_id", tracerID.String()))}
	}

	return l
}

func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.zap.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.zap.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.zap.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.zap.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zapcore.Field) {
	l.zap.Fatal(msg, fields...)
}

func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{zap: l.zap.With(zap.Error(err))}
}

func (l *Logger) Sync() error {
	return l.zap.Sync()
}

var defaultLogger *Logger

func InitDefault(config Config) {
	logger, _ := New(config)

	defaultLogger = logger
}

func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = &Logger{zap: zap.NewExample()}
	}
	return defaultLogger
}

func WithContext(ctx context.Context) *Logger {
	return Default().WithContext(ctx)
}

