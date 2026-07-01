package auth

import (
	"context"
	"errors"
	"log/slog"
	apierrors "mrtutor/api/errors"
	"testing"
	"time"
)

//go:generate go tool moq -out service_mock_test.go . principalRepository sessionStore

func TestService(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		t.Parallel()
		repo := &principalRepositoryMock{
			CreatePrincipalFunc: func(ctx context.Context, principal Principal) (*Principal, error) {
				return new(principal), nil
			},
		}
		store := &sessionStoreMock{
			CreateSessionFunc: func(ctx context.Context, userId int64) (*Session, error) {
				return new(Session{
					PrincipalID: userId,
					Token:       "sessionidtokenforuser",
				}), nil
			},
		}
		svc := serviceImpl{
			repository:   repo,
			sessionStore: store,
			logger:       slog.Default(),
		}

		t.Run("Can register", func(t *testing.T) {
			if _, err := svc.Register(context.Background(), RegisterRequest{
				Email:    "testuser@example.com",
				Username: "testuser",
				Password: "Password123!",
			}); err != nil {
				t.Fatal(err)
			}
		})

		t.Run("Cannot register when invalid", func(t *testing.T) {
			basePrincipal := RegisterRequest{
				Username: "me",
				Password: "Password123!",
				Email:    "me@example.com",
			}

			tt := []struct {
				name  string
				input RegisterRequest
			}{
				{
					name: "password too short",
					input: func() RegisterRequest {
						req := basePrincipal
						req.Password = "short1!"
						return req
					}(),
				},
				{
					name: "password missing uppercase",
					input: func() RegisterRequest {
						req := basePrincipal
						req.Password = "password123!"
						return req
					}(),
				},
				{
					name: "password missing lowercase",
					input: func() RegisterRequest {
						req := basePrincipal
						req.Password = "PASSWORD123!"
						return req
					}(),
				},
				{
					name: "password missing number",
					input: func() RegisterRequest {
						req := basePrincipal
						req.Password = "Password!"
						return req
					}(),
				},
				{
					name: "password missing special character",
					input: func() RegisterRequest {
						req := basePrincipal
						req.Password = "Password123"
						return req
					}(),
				},
			}

			for _, tc := range tt {
				t.Run(tc.name, func(t *testing.T) {
					if _, err := svc.Register(context.Background(), tc.input); err == nil {
						t.Fatal("expected error but got nil")
					}
				})
			}
		})
	})

	t.Run("Login", func(t *testing.T) {
		t.Parallel()
		password := "Password123!"
		passwordHash, err := HashPassword(password)
		if err != nil {
			t.Fatal(err)
		}

		persistedPrincipal := Principal{
			ID:             1,
			Username:       "testuser",
			HashedPassword: passwordHash,
			Email:          "testuser@example.com",
		}
		repo := &principalRepositoryMock{
			FindPrincipalByEmailOrUsernameFunc: func(ctx context.Context, token string) (*Principal, error) {
				if token == persistedPrincipal.Email || token == persistedPrincipal.Username {
					return &persistedPrincipal, nil
				}
				return nil, errPrincipalNotFound
			},
		}
		store := &sessionStoreMock{
			CreateSessionFunc: func(ctx context.Context, userId int64) (*Session, error) {
				return new(Session{
					PrincipalID: userId,
					Token:       "sessionidtokenforuser",
				}), nil
			},
		}
		svc := serviceImpl{
			repository:   repo,
			sessionStore: store,
			logger:       slog.Default(),
		}

		t.Run("Can login by email", func(t *testing.T) {
			if sessionToken, err := svc.Login(context.Background(), LoginRequest{
				Token:    persistedPrincipal.Email,
				Password: password,
			}); err != nil {
				t.Fatal(err)
			} else if sessionToken != "sessionidtokenforuser" {
				t.Fatalf("expected session token 'sessionidtokenforuser' but got '%s'", sessionToken)
			}
		})

		t.Run("Can login by username", func(t *testing.T) {
			if sessionToken, err := svc.Login(context.Background(), LoginRequest{
				Token:    persistedPrincipal.Username,
				Password: password,
			}); err != nil {
				t.Fatal(err)
			} else if sessionToken != "sessionidtokenforuser" {
				t.Fatalf("expected session token 'sessionidtokenforuser' but got '%s'", sessionToken)
			}
		})

		t.Run("Unauthorized when password not correct", func(t *testing.T) {
			if _, err := svc.Login(context.Background(), LoginRequest{
				Token:    persistedPrincipal.Username,
				Password: "WrongPassword123!",
			}); err == nil {
				t.Fatal("Expected error got nil")
			} else if !errors.Is(err, apierrors.ErrUnauthorized) {
				t.Fatalf("Expected error to ErrUnauthorized but got %v", err)
			}
		})

		t.Run("Wrong token", func(t *testing.T) {
			if _, err := svc.Login(context.Background(), LoginRequest{
				Token:    "nonexistentuser",
				Password: password,
			}); err == nil {
				t.Fatal("Expected error got nil")
			} else if !errors.Is(err, apierrors.ErrUnauthorized) {
				t.Fatalf("Expected error to ErrUnauthorized but got %v", err)
			}
		})
	})

	t.Run("VerifySession", func(t *testing.T) {
		t.Parallel()
		const validToken = "valid-session-token"
		principal := Principal{ID: 1, Username: "testuser", Email: "testuser@example.com"}

		t.Run("Returns principal for a valid session", func(t *testing.T) {
			store := &sessionStoreMock{
				GetSessionFunc: func(ctx context.Context, sessionToken string) (*Principal, error) {
					return &principal, nil
				},
				RefreshSessionFunc: func(ctx context.Context, sessionToken string) (time.Time, error) {
					return time.Time{}, nil
				},
			}
			svc := serviceImpl{sessionStore: store, logger: slog.Default()}

			got, err := svc.VerifySession(context.Background(), validToken)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got == nil || got.ID != principal.ID {
				t.Fatalf("expected principal %+v, got %+v", principal, got)
			}
		})

		t.Run("Unauthorized when session not found", func(t *testing.T) {
			store := &sessionStoreMock{
				GetSessionFunc: func(ctx context.Context, sessionToken string) (*Principal, error) {
					return nil, errSessionNotFound
				},
			}
			svc := serviceImpl{sessionStore: store, logger: slog.Default()}

			if _, err := svc.VerifySession(context.Background(), "missing"); !errors.Is(err, apierrors.ErrUnauthorized) {
				t.Fatalf("expected ErrUnauthorized, got %v", err)
			}
		})

		t.Run("Propagates unexpected store errors", func(t *testing.T) {
			storeErr := errors.New("db unavailable")
			store := &sessionStoreMock{
				GetSessionFunc: func(ctx context.Context, sessionToken string) (*Principal, error) {
					return nil, storeErr
				},
			}
			svc := serviceImpl{sessionStore: store, logger: slog.Default()}

			principal, err := svc.VerifySession(context.Background(), validToken)
			if !errors.Is(err, storeErr) {
				t.Fatalf("expected store error to propagate, got %v", err)
			}
			if principal != nil {
				t.Fatalf("expected nil principal on error, got %+v", principal)
			}
		})
	})
}
