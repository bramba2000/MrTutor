package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"mrtutor-api/db/queries"
	apierrors "mrtutor-api/errors"
	"time"
)

const sessionDuration = 24 * time.Hour
const sessionIdleDuration = 30 * time.Minute

type sqlRepository struct {
	db      *sql.DB
	queries *queries.Queries
}

var (
	errSessionNotFound   = apierrors.NotFoundError{Entity: "session"}
	errPrincipalNotFound = apierrors.NotFoundError{Entity: "principal"}
)

// CreatePrincipal implements [principalRepository].
func (s *sqlRepository) CreatePrincipal(ctx context.Context, principal Principal) (*Principal, error) {
	userModel, err := s.queries.CreateUser(ctx, PrincipalToCreateUserParam(principal))
	if err != nil {
		return nil, err
	}
	return new(UserToPrincipal(userModel)), nil
}

// FindPrincipalByEmailOrUsername implements [principalRepository].
func (s *sqlRepository) FindPrincipalByEmailOrUsername(ctx context.Context, token string) (*Principal, error) {
	userModel, err := s.queries.GetUserByEmailOrUsername(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errPrincipalNotFound
		}
		return nil, err
	}
	return new(UserToPrincipal(userModel)), nil
}

func newSQLRepository(db *sql.DB) principalRepository {
	return &sqlRepository{
		db:      db,
		queries: queries.New(db),
	}
}

type sqlSessionStore struct {
	db      *sql.DB
	queries *queries.Queries
}

// CreateSession implements [sessionStore].
func (s *sqlSessionStore) CreateSession(ctx context.Context, principalID int64) (*Session, error) {
	now := time.Now()
	result, err := s.queries.CreateSession(ctx, queries.CreateSessionParams{
		UserID:         principalID,
		Token:          rand.Text(),
		AbsoluteExpiry: now.Add(sessionDuration),
		IdleExpiry:     now.Add(sessionIdleDuration),
	})
	if err != nil {
		return nil, err
	}
	return &Session{
		Token:       result.Token,
		PrincipalID: result.UserID,
	}, nil
}

// DeleteSession implements [sessionStore].
func (s *sqlSessionStore) DeleteSession(ctx context.Context, sessionToken string) error {
	return s.queries.DeleteSession(ctx, sessionToken)
}

// GetSession implements [sessionStore].
func (s *sqlSessionStore) GetSession(ctx context.Context, sessionToken string) (*Principal, error) {
	sessionModel, err := s.queries.GetUserBySessionId(ctx, sessionToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errSessionNotFound
		}
		return nil, err
	}
	return new(UserToPrincipal(sessionModel)), nil
}

// RefreshSession implements [sessionStore].
func (s *sqlSessionStore) RefreshSession(ctx context.Context, sessionToken string) (time.Time, error) {
	expire := time.Now().Add(sessionIdleDuration)
	err := s.queries.UpdateSessionIdleExpiry(ctx, queries.UpdateSessionIdleExpiryParams{
		Token:      sessionToken,
		IdleExpiry: expire,
	})
	return expire, err
}

func newSQLSessionStore(db *sql.DB) sessionStore {
	return &sqlSessionStore{
		db:      db,
		queries: queries.New(db),
	}
}
