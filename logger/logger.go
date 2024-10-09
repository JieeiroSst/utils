package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ConfigZap() *zap.SugaredLogger {
	cfg := zap.Config{
		Encoding:    "json",
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		OutputPaths: []string{"stderr"},

		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",
			TimeKey:      "time",
			LevelKey:     "level",
			CallerKey:    "caller",
			EncodeCaller: zapcore.FullCallerEncoder,
			EncodeLevel:  CustomLevelEncoder,
			EncodeTime:   SyslogTimeEncoder,
		},
	}

	logger, _ := cfg.Build()
	return logger.Sugar()
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func Info(msg interface{}) {
	ConfigZap().Info(msg)
}

func Error(msg interface{}) {
	ConfigZap().Error(msg)
}

func Warn(msg interface{}) {
	ConfigZap().Warn(msg)
}

func Debug(msg interface{}) {
	ConfigZap().Debug(msg)
}
