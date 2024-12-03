package logger

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Key = "TRANSACTION_ID_KEY"
)

func generateTransactionID() string {
	return uuid.New().String()
}

func TracerID(ctx context.Context) context.Context {
	tid := generateTransactionID()
	return context.WithValue(ctx, Key, tid)
}

func ToString(ctx context.Context) string {
	tid := ctx.Value(Key)
	return tid.(string)
}

func ConfigZap(ctx context.Context) *zap.Logger {
	ctx = TracerID(ctx)
	config := zap.NewProductionConfig()
	logger, _ := config.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(zapcore.AddSync(zapcore.Lock(os.Stdout))),
			zapcore.InfoLevel,
		).With([]zapcore.Field{zap.String("transaction_id", ToString(ctx))}))
	}))
	return logger
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func Info(ctx context.Context, msg string, args ...interface{}) {
	logger := ConfigZap(ctx)
	logger.Sugar().Infof(msg, args...)
}

func Error(ctx context.Context, msg string, args ...interface{}) {
	logger := ConfigZap(ctx)
	logger.Sugar().Infof(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...interface{}) {
	logger := ConfigZap(ctx)
	logger.Sugar().Infof(msg, args...)
}

func Debug(ctx context.Context, msg string, args ...interface{}) {
	logger := ConfigZap(ctx)
	logger.Sugar().Infof(msg, args...)
}
