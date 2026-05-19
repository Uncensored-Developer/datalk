package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func requestLoggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError:  true,
		LogLatency:   true,
		LogMethod:    true,
		LogURIPath:   true,
		LogRequestID: true,
		LogStatus:    true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.String("method", values.Method),
				slog.String("path", values.URIPath),
				slog.String("route", values.URIPath),
				slog.Int("status", values.Status),
				slog.Int64("latency_ms", values.Latency.Milliseconds()),
				slog.String("request_id", values.RequestID),
			}
			if values.Error != nil {
				attrs = append(attrs, slog.Any("err", values.Error))
			}

			level := slog.LevelInfo
			if values.Status >= http.StatusInternalServerError {
				level = slog.LevelError
			} else if values.Status >= http.StatusBadRequest {
				level = slog.LevelWarn
			}

			logger.LogAttrs(context.Background(), level, "http request completed", attrs...)
			return nil
		},
	})
}
