package main

import (
	"context"
	"errors"
	"fmt"
	"mrtutor-api/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

var isShuttingDownServer atomic.Bool

func newServer(ctx context.Context) *http.Server {
	mux := http.NewServeMux()
	addRoutes(mux)

	handler := applyMiddleware(mux, loggingMiddleware)

	return &http.Server{
		Addr:        net.JoinHostPort("", config.Port),
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     handler,
	}
}

func startServer(server *http.Server) {
	fmt.Println("Starting server at " + server.Addr)
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Server error: %v\n", err)
	}
}

func shutdownServer(server *http.Server, stopRequestContext context.CancelFunc) {
	isShuttingDownServer.Store(true)
	fmt.Println("Shutting down server...")

	shutdownContext, stopShutdown := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer stopShutdown()

	if err := server.Shutdown(shutdownContext); err != nil {
		fmt.Printf("Error during server shutdown: %v\n", err)
	} else {
		fmt.Println("Server shutdown completed successfully.")
	}
	stopRequestContext()
}

func runServer(ctx context.Context, stop context.CancelFunc) {
	ongoingCtx, stopOngoingGracefully := context.WithCancel(context.Background())
	server := newServer(ongoingCtx)

	go startServer(server)

	<-ctx.Done()
	stop()
	shutdownServer(server, stopOngoingGracefully)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runServer(ctx, stop)
}
