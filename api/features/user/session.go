package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"mrtutor-api/db/queries"
	"time"
)

const sessionDuration = 24 * time.Hour
const sessionIdleDuration = 30 * time.Minute

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

func wrapSessionError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSessionNotFound
	}
	return err
}

type Session struct {
	queries.Session
}

func (s Session) toCreateParams() queries.CreateSessionParams {
	return queries.CreateSessionParams{
		UserID:         s.UserID,
		Token:          s.Token,
		AbsoluteExpiry: s.AbsoluteExpiry,
		IdleExpiry:     s.IdleExpiry,
	}
}

type SessionStore interface {
	// CreateSession creates a new session for the given user ID and returns the session ID.
	CreateSession(context.Context, Session) (Session, error)
	// CleanupExpiredSessions deletes all sessions that have expired from the store.
	CleanupExpiredSessions(context.Context)
	// GetSession retrieves a session by its session ID and checks if it is valid (not expired).
	GetSession(context.Context, string) (Session, error)
	// UpdateSessionIdleExpiry updates the idle expiry time of a session to extend its validity.
	UpdateSessionIdleExpiry(context.Context, string, time.Time) error
}

type sqlSessionStore struct {
	db      *sql.DB
	queries *queries.Queries
}

// CleanupExpiredSessions implements [SessionStore].
func (s sqlSessionStore) CleanupExpiredSessions(ctx context.Context) {
	s.queries.DeleteExpiredSessions(ctx)
}

// CreateSession implements [SessionStore].
func (s sqlSessionStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	sess, err := s.queries.CreateSession(ctx, session.toCreateParams())
	return Session{Session: sess}, err
}

// UpdateSessionIdleExpiry implements [SessionStore].
func (s sqlSessionStore) GetSession(ctx context.Context, sessionID string) (Session, error) {
	sess, err := s.queries.GetSessionById(ctx, sessionID)
	return Session{Session: sess}, wrapSessionError(err)
}

// UpdateSessionIdleExpiry implements [SessionStore].
func (s sqlSessionStore) UpdateSessionIdleExpiry(ctx context.Context, sessionID string, newIdleExpiry time.Time) error {
	return s.queries.UpdateSessionIdleExpiry(ctx, queries.UpdateSessionIdleExpiryParams{
		Token:      sessionID,
		IdleExpiry: newIdleExpiry,
	})
}

func newSessionStore(db *sql.DB) SessionStore {
	return sqlSessionStore{
		db:      db,
		queries: queries.New(db),
	}
}

type SessionManager interface {
	// CreateSession creates a new session for the given user ID and returns the session.
	CreateSession(context.Context, int64) (Session, error)
	// CleanupExpiredSessions deletes all sessions that have expired from the store.
	CleanupExpiredSessions(context.Context)
	// CheckValidSession checks if the given session ID is valid and returns the associated session if it is.
	CheckValidSession(context.Context, string) (Session, error)
}

type sessionManager struct {
	store SessionStore
}

func (m sessionManager) CreateSession(ctx context.Context, userID int64) (Session, error) {
	now := time.Now()
	session := Session{
		Session: queries.Session{
			UserID:         userID,
			Token:          rand.Text(),
			AbsoluteExpiry: now.Add(sessionDuration),
			IdleExpiry:     now.Add(sessionIdleDuration),
		},
	}
	return m.store.CreateSession(ctx, session)
}

func (m sessionManager) CleanupExpiredSessions(ctx context.Context) {
	m.store.CleanupExpiredSessions(ctx)
}

func (m sessionManager) CheckValidSession(ctx context.Context, sessionID string) (Session, error) {
	session, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	if now.After(session.AbsoluteExpiry) || now.After(session.IdleExpiry) {
		return Session{}, ErrSessionExpired
	}
	go func() {
		// Update idle expiry in the background to avoid blocking the request.
		_ = m.store.UpdateSessionIdleExpiry(context.WithoutCancel(ctx), sessionID, now.Add(sessionIdleDuration))
	}()
	return session, nil
}

func NewSessionManager(store SessionStore) SessionManager {
	return sessionManager{store: store}
}
