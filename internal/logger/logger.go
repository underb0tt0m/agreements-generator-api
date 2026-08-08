package logger

import (
	"fmt"

	"agreements-generator/internal/config"

	"go.uber.org/zap"
)

const (
	FieldError string = "error"
)

type Logger interface {
	Fatal(msg string, fields ...any)
	Error(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
}

func Load(cfg *config.Config) (Logger, error) {
	switch cfg.Log.Type {
	case "zap":
		return newZap(cfg)
	default:
		return newZap(cfg)
	}
}

type zapLogger struct {
	l *zap.Logger
}

func newZap(cfg *config.Config) (*zapLogger, error) {
	switch cfg.Log.Level {
	case "production":
		l, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("can't init logger: %w", err)
		}
		return &zapLogger{l: l}, nil
	case "development":
		l, err := zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("can't init logger: %w", err)
		}
		return &zapLogger{l: l}, nil
	default:
		l, err := zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("can't init logger: %w", err)
		}
		return &zapLogger{l: l}, nil
	}
}

func (l *zapLogger) Fatal(msg string, fields ...any) {
	l.l.Fatal(msg, toZapFields(fields...)...)
}

func (l *zapLogger) Error(msg string, fields ...any) {
	l.l.Error(msg, toZapFields(fields...)...)
}

func (l *zapLogger) Warn(msg string, fields ...any) {
	l.l.Warn(msg, toZapFields(fields...)...)
}

func (l *zapLogger) Debug(msg string, fields ...any) {
	l.l.Debug(msg, toZapFields(fields...)...)
}

func (l *zapLogger) Info(msg string, fields ...any) {
	l.l.Info(msg, toZapFields(fields...)...)
}

func toZapFields(args ...any) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(args)/2)

	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}

		if i+1 >= len(args) {
			fields = append(fields, zap.String(key, "MISSING_VALUE"))
		}

		val := args[i+1]
		switch v := val.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case int64:
			fields = append(fields, zap.Int64(key, v))
		case float64:
			fields = append(fields, zap.Float64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		case error:
			fields = append(fields, zap.Error(v)) //TODO ну дичь какая-то, переделать надо логгер
		case zap.Field:
			fields = append(fields, v)
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}
	return fields
}
