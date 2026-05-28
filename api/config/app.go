package config

import (
	"cmp"
	"log/slog"
	"os"
	"strings"
)

type AppMode string

const (
	DEV  AppMode = "dev"
	PROD AppMode = "prod"
	TEST AppMode = "test"
)

var (
	Mode     AppMode    = cmp.Or(parseAppMode(os.Getenv("APP_ENV")), DEV)
	LogLevel slog.Level = cmp.Or(parseLogLevel(os.Getenv("LOG_LEVEL")), slog.LevelInfo)
)

func parseAppMode(mode string) AppMode {
	mode = strings.ToLower(mode)
	switch mode {
	case "dev":
	case "development":
		return DEV
	case "prod":
	case "production":
		return PROD
	case "test":
	case "testing":
		return TEST
	default:
		return ""
	}
	return ""
}

func parseLogLevel(level string) slog.Level {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
