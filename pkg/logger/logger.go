package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

type Logger struct {
	slogger *slog.Logger
}

var defaultLogger *Logger

func InitLogger() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	slogger := slog.New(handler)
	defaultLogger = &Logger{slogger: slogger}
}

func (l *Logger) log(level slog.Level, msg string, args ...any) {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		file = filepath.Base(file)
		// Format the source as "file:line"
		source := fmt.Sprintf("%s:%d", file, line)
		args = append(args, slog.String("source", source))
	}

	l.slogger.Log(context.Background(), level, msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.log(slog.LevelInfo, msg, args...)
}

func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, slog.Any("error", err))
	}
	defaultLogger.log(slog.LevelError, msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.log(slog.LevelWarn, msg, args...)
}

func Fatal(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, slog.Any("error", err))
	}
	defaultLogger.log(slog.LevelError, msg, args...)
	os.Exit(1)
}
