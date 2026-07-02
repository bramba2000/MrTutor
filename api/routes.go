package main

import (
	"database/sql"
	"log/slog"
	"mrtutor/api/config"
	"mrtutor/api/features/auth"
	"mrtutor/api/scheduler"
	"mrtutor/api/static"
	"net/http"
	"strings"
)

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isShuttingDownServer.Load() {
			http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func addRoutes(mux *http.ServeMux, logger *slog.Logger, db *sql.DB, sched *scheduler.Scheduler) {
	internalMux := http.NewServeMux()
	internalMux.Handle("/health", healthHandler())

	basePath := strings.TrimRight(config.ApiBasePath, "/")
	if basePath == "" {
		basePath = "/"
	}

	if basePath == "/" {
		mux.Handle("/", internalMux)
		return
	}

	auth.InitModule(db, logger, sched).RegisterRoutes(internalMux)

	// Apply global middleware to all routes under the base path
	handler := applyMiddleware(
		http.StripPrefix(basePath, internalMux),
		newLoggingMiddleware(logger),
	)
	mux.Handle(basePath+"/", handler)
	// SPA fallback route to serve static files for any unmatched routes under the base path
	mux.Handle("/", static.Handler())
}
