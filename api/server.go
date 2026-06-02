package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"mrtutor/api/config"
	"net"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/go-co-op/gocron"
)

var isShuttingDownServer atomic.Bool

// newServer creates and configures a new HTTP server with the necessary routes and middleware.
// It returns the configured server and a cancel function to close the server's base context when needed.
func newServer(logger *slog.Logger, db *sql.DB, scheduler *gocron.Scheduler) (*http.Server, context.CancelFunc) {
	ctx, stop := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	addRoutes(mux, logger, db, scheduler)

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

// shutdownServer gracefully shuts down the HTTP server, allowing in-flight requests to complete before stopping the server.
func shutdownServer(logger *slog.Logger, server *http.Server, stopRequestContext context.CancelFunc) {
	isShuttingDownServer.Store(true)
	logger.Info("Initiating server shutdown")

	shutdownContext, stopShutdown := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer stopShutdown()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("Server shutdown error", "error", err)
		os.Exit(ServerShutdownErrorExitCode)
	} else {
		logger.Info("Server shutdown complete")
	}
	stopRequestContext()
}
