package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"mrtutor/api/config"
	"mrtutor/api/db"
	"mrtutor/api/db/migrations"
	"mrtutor/api/scheduler"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
)

const (
	SetupServerErrorExitCode = iota + 1
	ServerClosedUnexpectedlyExitCode
	ServerShutdownErrorExitCode
	SetupDbErrorExitCode
	FatalTaskErrorExitCode
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
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := newLogger()
	logger.Debug("starting server", "mode", config.Mode)

	// Register dependencies
	db := setupDb(logger)
	defer db.Close()

	// appCtx is cancelled by an OS signal OR by a fatal scheduled job, so both
	// converge on the single shutdown path below.
	appCtx, cancelApp := context.WithCancelCause(signalCtx)
	defer cancelApp(nil)

	sched := scheduler.New(logger,
		scheduler.WithLocation(time.UTC),
		scheduler.OnFatal(func(err error) { cancelApp(err) }),
	)
	server, cancelServerCtx := newServer(logger, db, sched)

	sched.Start(appCtx)
	go startServer(logger, server)

	<-appCtx.Done()
	stop()

	var fatal *scheduler.FatalError
	isFatal := errors.As(context.Cause(appCtx), &fatal)
	if isFatal {
		logger.Error("shutting down due to fatal task error", "error", context.Cause(appCtx))
	}

	// Stop background jobs first, then drain in-flight HTTP requests.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()
	if err := sched.Shutdown(shutdownCtx); err != nil {
		logger.Error("scheduler shutdown error", "error", err)
	}
	shutdownServer(logger, server, cancelServerCtx)

	if isFatal {
		os.Exit(FatalTaskErrorExitCode)
	}
}
