package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"mrtutor/api/db/queries"
	"mrtutor/api/scheduler"
	"mrtutor/api/validation"
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

type RegisterResponse struct {
	Principal
	SessionToken string
}

func (p Principal) VerifyPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.HashedPassword), []byte(password)) == nil
}

type RegisterRequest struct {
	Username string
	Email    string
	Password string
}

func (r RegisterRequest) Validate() error {
	problems := []string{}
	if !validation.Required(r.Username) {
		problems = append(problems, "username is required")
	} else if validation.IsValidEmail(r.Username) {
		problems = append(problems, "username cannot be an email")
	}
	if !validation.Required(r.Email) {
		problems = append(problems, "email is required")
	} else if !validation.IsValidEmail(r.Email) {
		problems = append(problems, "email is not valid")
	}
	if !validation.Required(r.Password) {
		problems = append(problems, "password is required")
	} else if msg := validation.IsValidPassword(r.Password); msg != "" {
		problems = append(problems, msg)
	}
	if len(problems) > 0 {
		return &validation.Error{
			Problems: problems,
		}
	}
	return nil
}

type LoginRequest struct {
	// Token represent either username or password for login attempt
	Token    string
	Password string
}

func (r LoginRequest) Validate() error {
	problems := []string{}
	if !validation.Required(r.Token) {
		problems = append(problems, "username or email is required")
	}
	if !validation.Required(r.Password) {
		problems = append(problems, "password is required")
	}
	if len(problems) > 0 {
		return &validation.Error{
			Problems: problems,
		}
	}
	return nil
}

// principalRepository defines the storage interface for managing principals
type principalRepository interface {
	CreatePrincipal(ctx context.Context, principal Principal) (*Principal, error)
	FindPrincipalByEmailOrUsername(ctx context.Context, token string) (*Principal, error)
}

type Session struct {
	Token       string
	PrincipalID int64
}

type sessionStore interface {
	GetSession(ctx context.Context, sessionToken string) (*Principal, error)
	CreateSession(ctx context.Context, principalID int64) (*Session, error)
	RefreshSession(ctx context.Context, sessionToken string) (time.Time, error)
	DeleteSession(ctx context.Context, sessionToken string) error
}

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (string, error)
	Logout(ctx context.Context, sessionToken string) error
	VerifySession(ctx context.Context, sessionToken string) (*Principal, error)
}

type module interface {
	Service
	RegisterRoutes(mux *http.ServeMux)
	// RequireAuth is middleware that other feature modules wrap their protected
	// handlers with to enforce authentication and inject the principal.
	RequireAuth(next http.Handler) http.Handler
}

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
