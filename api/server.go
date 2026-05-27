package main

import (
	"context"
	"errors"
	"fmt"
	"mrtutor-api/config"
	"net"
	"net/http"
	"os"
	"sync/atomic"
)

var isShuttingDownServer atomic.Bool

// newServer creates and configures a new HTTP server with the necessary routes and middleware.
// It returns the configured server and a cancel function to close the server's base context when needed.
func newServer() (*http.Server, context.CancelFunc) {
	ctx, stop := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	addRoutes(mux)

	handler := applyMiddleware(mux, loggingMiddleware)

	return &http.Server{
		Addr:        net.JoinHostPort("", config.Port),
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     handler,
	}, stop
}

// startServer starts the HTTP server and listens for incoming requests.
func startServer(server *http.Server) {
	fmt.Println("Starting server at " + server.Addr)
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(ServerClosedUnexpectedlyExitCode)
	}
}

// shutdownServer gracefully shuts down the HTTP server, allowing in-flight requests to complete before stopping the server.
func shutdownServer(server *http.Server, stopRequestContext context.CancelFunc) {
	isShuttingDownServer.Store(true)
	fmt.Println("Shutting down server...")

	shutdownContext, stopShutdown := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer stopShutdown()

	if err := server.Shutdown(shutdownContext); err != nil {
		fmt.Printf("Error during server shutdown: %v\n", err)
		os.Exit(ServerShutdownErrorExitCode)
	} else {
		fmt.Println("Server shutdown completed successfully.")
	}
	stopRequestContext()
}
