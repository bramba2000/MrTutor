package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"mrtutor/api/config"
	"mrtutor/api/scheduler"
	"net"
	"net/http"
	"os"
	"sync/atomic"
)

var isShuttingDownServer atomic.Bool

// newServer creates and configures a new HTTP server with the necessary routes and middleware.
// It returns the configured server and a cancel function to close the server's base context when needed.
func newServer(logger *slog.Logger, db *sql.DB, sched *scheduler.Scheduler) (*http.Server, context.CancelFunc) {
	ctx, stop := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	addRoutes(mux, logger, db, sched)

	return &http.Server{
		Addr:        net.JoinHostPort("", config.Port),
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     mux,
	}, stop
}

// startServer starts the HTTP server and listens for incoming requests.
func startServer(logger *slog.Logger, server *http.Server) {
	logger.Info("Starting server", "addr", server.Addr)
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Server error", "error", err)
		os.Exit(ServerClosedUnexpectedlyExitCode)
	}
}

// shutdownServer gracefully shuts down the HTTP server, allowing in-flight requests to
// complete before stopping the server. It drains within the deadline of the provided
// context, which is shared with the rest of the shutdown sequence.
func shutdownServer(ctx context.Context, logger *slog.Logger, server *http.Server, stopRequestContext context.CancelFunc) {
	isShuttingDownServer.Store(true)
	logger.Info("Initiating server shutdown")

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", "error", err)
		os.Exit(ServerShutdownErrorExitCode)
	} else {
		logger.Info("Server shutdown complete")
	}
	stopRequestContext()
}
