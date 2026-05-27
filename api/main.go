package main

import (
	"context"
	"database/sql"
	"fmt"
	"mrtutor-api/db"
	"os"
	"os/signal"
	"syscall"
)

const (
	SetupServerErrorExitCode = iota + 1
	ServerClosedUnexpectedlyExitCode
	ServerShutdownErrorExitCode
	SetupDbErrorExitCode
)

func setupDb() *sql.DB {
	// Initialize the database connection
	db, err := db.New()
	if err != nil {
		fmt.Errorf("failed to set up database: %v", err)
		os.Exit(SetupDbErrorExitCode)
	}
	return db
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Register dependencies
	server, cancelServerCtx := newServer()
	db := setupDb()

	// Start the server in a separate goroutine
	go startServer(server)

	<-ctx.Done()
	// Clean up resources and gracefully shut down the server
	stop()
	shutdownServer(server, cancelServerCtx)
}
