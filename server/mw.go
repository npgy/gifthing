package main

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func applyLoggingMiddleware(e *echo.Echo, logger *Logger) {
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogLatency:  true,
		LogError:    true,
		LogMethod:   true,
		LogRemoteIP: true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				{
					Key:   "URI",
					Value: slog.StringValue(v.URI),
				},
				{
					Key:   "IP",
					Value: slog.StringValue(v.RemoteIP),
				},
				{
					Key:   "Status",
					Value: slog.IntValue(v.Status),
				},
				{
					Key:   "Latency",
					Value: slog.StringValue(v.Latency.String()),
				},
				{
					Key:   "Method",
					Value: slog.StringValue(v.Method),
				},
			}

			if v.Error != nil {
				attrs = append(attrs, slog.Attr{
					Key:   "Error",
					Value: slog.StringValue(v.Error.Error()),
				})
				logger.LogAttrs(context.Background(), slog.LevelError, "REQUEST_ERROR", attrs...)
			} else {
				logger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST", attrs...)
			}

			return nil
		},
	}))
}
