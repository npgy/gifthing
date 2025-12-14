package main

import (
	"context"
	"log/slog"
	"os"
)

type Logger struct {
	stdoutLogger *slog.Logger
	fileLogger   *slog.Logger
	logFile      *os.File
}

func NewLogger() (*Logger, error) {
	l := &Logger{}
	logFile, err := os.OpenFile("gifthing.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return l, err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	fileLogger := slog.New(slog.NewJSONHandler(logFile, nil))

	l.fileLogger = fileLogger
	l.stdoutLogger = logger
	l.logFile = logFile
	return l, nil
}

func (l *Logger) LogSimple(level slog.Level, msg string, details string) {
	slogAttrs := []slog.Attr{
		slog.String("details", details),
	}
	l.LogAttrs(context.Background(), level, msg, slogAttrs...)
}

func (l *Logger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	l.stdoutLogger.LogAttrs(ctx, level, msg, attrs...)
	l.fileLogger.LogAttrs(ctx, level, msg, attrs...)
}
