package user

import (
	"context"
	"errors"
	"log/slog"
	apierrors "mrtutor-api/errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestLogin(t *testing.T) {
	password := "password"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	storedUser := User{
		ID:       1,
		Email:    "me@example.com",
		Username: "me",
		Password: string(hashedPassword),
	}

	repo := &mockUserRepository{
		GetUserByEmailOrUsernameFunc: func(ctx context.Context, emailOrUsername string) (User, error) {
			if emailOrUsername == storedUser.Email || emailOrUsername == storedUser.Username {
				return storedUser, nil
			}
			return User{}, ErrUserNotFound
		},
	}
	sessions := &mockSessionManager{
		CreateSessionFunc: func(_ context.Context, userID int64) (Session, error) {
			return Session{}, nil
		},
	}
	service := UserService(userService{repo: repo, logger: slog.Default(), sessionManager: sessions})

	t.Run("successful login by email", func(t *testing.T) {
		_, err := service.LoginUser(t.Context(), LoginUserParam{
			EmailOrUsername: storedUser.Email,
			Password:        password,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("successful login by username", func(t *testing.T) {
		_, err := service.LoginUser(t.Context(), LoginUserParam{
			EmailOrUsername: storedUser.Username,
			Password:        password,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := service.LoginUser(t.Context(), LoginUserParam{
			EmailOrUsername: storedUser.Email,
			Password:        "pswd00",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		} else if !errors.Is(err, apierrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("unexisting email", func(t *testing.T) {
		_, err := service.LoginUser(t.Context(), LoginUserParam{
			EmailOrUsername: storedUser.Email + "x",
			Password:        password,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		} else if !errors.Is(err, apierrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("unexisting username", func(t *testing.T) {
		_, err := service.LoginUser(t.Context(), LoginUserParam{
			EmailOrUsername: storedUser.Username + "x",
			Password:        password,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		} else if !errors.Is(err, apierrors.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

type mockUserRepository struct {
	CreateUserFunc               func(ctx context.Context, param CreateUserParams) (User, error)
	GetUserByEmailOrUsernameFunc func(ctx context.Context, emailOrUsername string) (User, error)
}

// CreateUser implements [UserRepository].
func (m *mockUserRepository) CreateUser(ctx context.Context, param CreateUserParams) (User, error) {
	return m.CreateUserFunc(ctx, param)
}

// GetUserByEmailOrUsername implements [UserRepository].
func (m *mockUserRepository) GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (User, error) {
	return m.GetUserByEmailOrUsernameFunc(ctx, emailOrUsername)
}

var _ UserRepository = (*mockUserRepository)(nil)

type mockSessionManager struct {
	CreateSessionFunc          func(ctx context.Context, userID int64) (Session, error)
	CleanupExpiredSessionsFunc func(ctx context.Context)
	CheckValidSessionFunc      func(ctx context.Context, sessionID string) (Session, error)
}

func (m *mockSessionManager) CreateSession(ctx context.Context, userID int64) (Session, error) {
	return m.CreateSessionFunc(ctx, userID)
}

func (m *mockSessionManager) CleanupExpiredSessions(ctx context.Context) {
	m.CleanupExpiredSessionsFunc(ctx)
}

func (m *mockSessionManager) CheckValidSession(ctx context.Context, sessionID string) (Session, error) {
	return m.CheckValidSessionFunc(ctx, sessionID)
}

var _ SessionManager = (*mockSessionManager)(nil)
