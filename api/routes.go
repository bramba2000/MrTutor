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
	// Health is served at the root, outside the API base path and its logging
	// middleware, so probes have a stable path and don't spam request logs.
	mux.Handle("/health", healthHandler())

	basePath := strings.TrimRight(config.ApiBasePath, "/")
	if basePath == "" {
		// No API base path configured: only the health endpoint is served.
		return
	}

	apiMux := http.NewServeMux()
	auth.InitModule(db, logger, sched).RegisterRoutes(apiMux)

	// Apply global middleware to all routes under the base path
	handler := applyMiddleware(
		http.StripPrefix(basePath, apiMux),
		newLoggingMiddleware(logger),
	)
	mux.Handle(basePath+"/", handler)
	// SPA fallback route to serve static files for any unmatched routes under the base path
	mux.Handle("/", static.Handler())
}
