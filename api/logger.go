package main

import (
	"log/slog"
	"mrtutor-api/config"
	"os"
)

func newLogger() *slog.Logger {
	removeTime := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 && config.Mode != config.PROD {
			return slog.Attr{}
		}
		return a
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:       config.LogLevel,
		ReplaceAttr: removeTime,
	})
	return slog.New(handler)
}
