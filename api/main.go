package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"mrtutor/api/config"
	"mrtutor/api/db"
	"mrtutor/api/db/migrations"
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

func setupDb(logger *slog.Logger) *sql.DB {
	var dbInstance *sql.DB
	var err error

	logger = logger.With("component", "setup")

	if config.Mode == config.TEST {
		dbInstance, err = db.NewInMemory()
	} else {
		dbInstance, err = db.New()
	}

	if err != nil {
		logger.Error("failed to set up database", "error", err)
		os.Exit(SetupDbErrorExitCode)
	}

	if config.Mode == config.DEV || config.Mode == config.TEST {
		m, err := migrations.NewWithDb(dbInstance)

		if err != nil {
			logger.Error("failed to create migration instance", "error", err)
			os.Exit(SetupDbErrorExitCode)
		}

		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.Error("failed to run database migrations", "error", err)
			os.Exit(SetupDbErrorExitCode)
		}
	}

	return dbInstance
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := newLogger()
	logger.Debug("starting server", "mode", config.Mode)

	// Register dependencies
	db := setupDb(logger)
	server, cancelServerCtx := newServer(logger, db)
	defer db.Close()

	// Start the server in a separate goroutine
	go startServer(logger, server)

	<-ctx.Done()
	// Clean up resources and gracefully shut down the server
	stop()
	shutdownServer(logger, server, cancelServerCtx)
}
