package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"mrtutor/api/db/queries"
	"mrtutor/api/scheduler"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Principal represent an authenticated entity in the system
type Principal struct {
	ID             int64
	Username       string
	Email          string
	HashedPassword string `json:"-"`
	CreateAt       time.Time
	ModifiedAt     time.Time
}

func (p Principal) VerifyPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.HashedPassword), []byte(password)) == nil
}

type Session struct {
	Token       string
	PrincipalID int64
}

type module interface {
	Service
	RegisterRoutes(mux *http.ServeMux)
	// RequireAuth is middleware that other feature modules wrap their protected
	// handlers with to enforce authentication and inject the principal.
	RequireAuth(next http.Handler) http.Handler
}

// Build the auth module, which includes the service, controller, and middleware. The module is
// initialized with the database connection, logger, and scheduler for background tasks.
func InitModule(db *sql.DB, logger *slog.Logger, sched *scheduler.Scheduler) module {
	queries := queries.New(db)
	sessionStore := &sqlSessionStore{db, queries}

	service := &serviceImpl{
		repository:   &sqlRepository{db, queries},
		sessionStore: sessionStore,
		logger:       logger,
	}

	// Hourly cleanup of expired sessions. The job uses the scheduler-provided
	// ctx (cancelled on shutdown) and returns its error so the runner logs it;
	// a failed cleanup is non-fatal and retries on the next hour.
	err := sched.Add("session-cleanup", scheduler.Periodic(time.Hour), func(ctx context.Context) error {
		if err := sessionStore.DeleteExpiredSessions(ctx); err != nil {
			return fmt.Errorf("delete expired sessions: %w", err)
		}
		return nil
	})

	if err != nil {
		panic(fmt.Sprintf("failed to schedule session cleanup: %v", err))
	}

	return struct {
		Service
		controller
	}{
		Service:    service,
		controller: controller{service: service},
	}
}
