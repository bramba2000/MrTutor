package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mrtutor-api/config"
	"mrtutor-api/db"
	"mrtutor-api/db/migrations"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
)

const (
	SetupServerErrorExitCode = iota + 1
	ServerClosedUnexpectedlyExitCode
	ServerShutdownErrorExitCode
	SetupDbErrorExitCode
)

func setupDb() *sql.DB {
	var dbInstance *sql.DB
	var err error

	if config.Mode == config.TEST {
		dbInstance, err = db.NewInMemory()
	} else {
		dbInstance, err = db.New()
	}

	if err != nil {
		fmt.Printf("failed to set up database: %v\n", err)
		os.Exit(SetupDbErrorExitCode)
	}

	if config.Mode == config.DEV || config.Mode == config.TEST {
		m, err := migrations.NewWithDb(dbInstance)

		if err != nil {
			fmt.Printf("failed to initialize database migrations: %v", err)
			os.Exit(SetupDbErrorExitCode)
		}

		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Printf("failed to run database migrations: %v", err)
			os.Exit(SetupDbErrorExitCode)
		}
	}

	return dbInstance
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Register dependencies
	server, cancelServerCtx := newServer()
	db := setupDb()
	defer db.Close()

	// Start the server in a separate goroutine
	go startServer(server)

	<-ctx.Done()
	// Clean up resources and gracefully shut down the server
	stop()
	shutdownServer(server, cancelServerCtx)
}
