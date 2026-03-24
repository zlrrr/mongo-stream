package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a production-style zap logger that writes JSON to stdout.
func New(debug bool) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			MessageKey:     "msg",
			CallerKey:      "caller",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}
	return cfg.Build()
}

// Must creates a logger and panics on error.
func Must(debug bool) *zap.Logger {
	l, err := New(debug)
	if err != nil {
		panic(err)
	}
	return l
}
