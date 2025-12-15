package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/lmittmann/tint"
)

func SetupLogger(cfg config.Config) *slog.Logger {
	handler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})
	if cfg.AppEnv == config.AppEnvProduction {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return slog.New(handler)
}

func Err(err error) slog.Attr {
	return slog.Any("err", err)
}

func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
