package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"mrtutor-api/db/queries"
)

type mockSessionStore struct {
	CreateSessionFunc             func(ctx context.Context, session Session) (Session, error)
	CleanupExpiredSessionsFunc    func(ctx context.Context)
	GetSessionFunc                func(ctx context.Context, sessionID string) (Session, error)
	UpdateSessionIdleExpiryFunc   func(ctx context.Context, sessionID string, newIdleExpiry time.Time) error
}

func (m *mockSessionStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	return m.CreateSessionFunc(ctx, session)
}

func (m *mockSessionStore) CleanupExpiredSessions(ctx context.Context) {
	m.CleanupExpiredSessionsFunc(ctx)
}

func (m *mockSessionStore) GetSession(ctx context.Context, sessionID string) (Session, error) {
	return m.GetSessionFunc(ctx, sessionID)
}

func (m *mockSessionStore) UpdateSessionIdleExpiry(ctx context.Context, sessionID string, newIdleExpiry time.Time) error {
	return m.UpdateSessionIdleExpiryFunc(ctx, sessionID, newIdleExpiry)
}

var _ SessionStore = (*mockSessionStore)(nil)

func TestSessionManager_CreateSession(t *testing.T) {
	const userID = int64(42)
	var capturedSession Session

	store := &mockSessionStore{
		CreateSessionFunc: func(_ context.Context, session Session) (Session, error) {
			capturedSession = session
			return session, nil
		},
	}
	manager := sessionManager{store: store}

	before := time.Now()
	result, err := manager.CreateSession(t.Context(), userID)
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != userID {
		t.Errorf("UserID: expected %d, got %d", userID, result.UserID)
	}
	if len(capturedSession.Token) == 0 {
		t.Error("expected non-empty token")
	}
	if capturedSession.AbsoluteExpiry.Before(before.Add(sessionDuration)) || capturedSession.AbsoluteExpiry.After(after.Add(sessionDuration)) {
		t.Errorf("AbsoluteExpiry %v not in expected range [%v, %v]", capturedSession.AbsoluteExpiry, before.Add(sessionDuration), after.Add(sessionDuration))
	}
	if capturedSession.IdleExpiry.Before(before.Add(sessionIdleDuration)) || capturedSession.IdleExpiry.After(after.Add(sessionIdleDuration)) {
		t.Errorf("IdleExpiry %v not in expected range [%v, %v]", capturedSession.IdleExpiry, before.Add(sessionIdleDuration), after.Add(sessionIdleDuration))
	}
}

func TestSessionManager_CreateSession_StoreError(t *testing.T) {
	wantErr := errors.New("db error")
	store := &mockSessionStore{
		CreateSessionFunc: func(_ context.Context, _ Session) (Session, error) {
			return Session{}, wantErr
		},
	}
	manager := sessionManager{store: store}

	_, err := manager.CreateSession(t.Context(), 1)
	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestSessionManager_CleanupExpiredSessions(t *testing.T) {
	called := false
	store := &mockSessionStore{
		CleanupExpiredSessionsFunc: func(_ context.Context) {
			called = true
		},
	}
	manager := sessionManager{store: store}

	manager.CleanupExpiredSessions(t.Context())

	if !called {
		t.Error("expected CleanupExpiredSessions to be called on store")
	}
}

func noopUpdateIdleExpiry(_ context.Context, _ string, _ time.Time) error { return nil }

func TestSessionManager_CheckValidSession(t *testing.T) {
	validSession := Session{
		Session: queries.Session{
			UserID:         1,
			Token:          "valid-token",
			AbsoluteExpiry: time.Now().Add(time.Hour),
			IdleExpiry:     time.Now().Add(time.Hour),
		},
	}

	t.Run("valid session", func(t *testing.T) {
		store := &mockSessionStore{
			GetSessionFunc:              func(_ context.Context, _ string) (Session, error) { return validSession, nil },
			UpdateSessionIdleExpiryFunc: noopUpdateIdleExpiry,
		}
		got, err := sessionManager{store: store}.CheckValidSession(t.Context(), validSession.Token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Token != validSession.Token {
			t.Errorf("expected token %q, got %q", validSession.Token, got.Token)
		}
	})

	t.Run("idle expiry is renewed on valid session", func(t *testing.T) {
		renewedCh := make(chan time.Time, 1)
		store := &mockSessionStore{
			GetSessionFunc: func(_ context.Context, _ string) (Session, error) { return validSession, nil },
			UpdateSessionIdleExpiryFunc: func(_ context.Context, _ string, newIdleExpiry time.Time) error {
				renewedCh <- newIdleExpiry
				return nil
			},
		}
		before := time.Now()
		_, err := sessionManager{store: store}.CheckValidSession(t.Context(), validSession.Token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		select {
		case renewed := <-renewedCh:
			if renewed.Before(before.Add(sessionIdleDuration)) {
				t.Errorf("renewed idle expiry %v is earlier than expected", renewed)
			}
		case <-time.After(time.Second):
			t.Error("idle expiry was not renewed")
		}
	})

	t.Run("session not found", func(t *testing.T) {
		store := &mockSessionStore{
			GetSessionFunc: func(_ context.Context, _ string) (Session, error) { return Session{}, ErrSessionNotFound },
		}
		_, err := sessionManager{store: store}.CheckValidSession(t.Context(), "nonexistent")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("absolute expiry exceeded", func(t *testing.T) {
		expired := Session{
			Session: queries.Session{
				UserID:         1,
				Token:          "expired-token",
				AbsoluteExpiry: time.Now().Add(-time.Hour),
				IdleExpiry:     time.Now().Add(time.Hour),
			},
		}
		store := &mockSessionStore{
			GetSessionFunc: func(_ context.Context, _ string) (Session, error) { return expired, nil },
		}
		_, err := sessionManager{store: store}.CheckValidSession(t.Context(), expired.Token)
		if !errors.Is(err, ErrSessionExpired) {
			t.Errorf("expected ErrSessionExpired, got %v", err)
		}
	})

	t.Run("idle expiry exceeded", func(t *testing.T) {
		expired := Session{
			Session: queries.Session{
				UserID:         1,
				Token:          "idle-expired-token",
				AbsoluteExpiry: time.Now().Add(time.Hour),
				IdleExpiry:     time.Now().Add(-time.Minute),
			},
		}
		store := &mockSessionStore{
			GetSessionFunc: func(_ context.Context, _ string) (Session, error) { return expired, nil },
		}
		_, err := sessionManager{store: store}.CheckValidSession(t.Context(), expired.Token)
		if !errors.Is(err, ErrSessionExpired) {
			t.Errorf("expected ErrSessionExpired, got %v", err)
		}
	})

	t.Run("store error is propagated", func(t *testing.T) {
		wantErr := errors.New("unexpected db error")
		store := &mockSessionStore{
			GetSessionFunc: func(_ context.Context, _ string) (Session, error) { return Session{}, wantErr },
		}
		_, err := sessionManager{store: store}.CheckValidSession(t.Context(), "any")
		if !errors.Is(err, wantErr) {
			t.Errorf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestWrapSessionError(t *testing.T) {
	t.Run("sql.ErrNoRows maps to ErrSessionNotFound", func(t *testing.T) {
		err := wrapSessionError(sql.ErrNoRows)
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("other errors pass through unchanged", func(t *testing.T) {
		original := errors.New("some db error")
		err := wrapSessionError(original)
		if !errors.Is(err, original) {
			t.Errorf("expected original error, got %v", err)
		}
	})
}
