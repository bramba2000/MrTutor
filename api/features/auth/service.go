package auth

import (
	"context"
	"errors"
	"log/slog"
	apierrors "mrtutor/api/errors"
	"mrtutor/api/validation"

	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (string, error)
	Logout(ctx context.Context, sessionToken string) error
	VerifySession(ctx context.Context, sessionToken string) (*Principal, error)
}

type RegisterResponse struct {
	Principal
	SessionToken string
}

type RegisterRequest struct {
	Username string
	Email    string
	Password string
}

func (r RegisterRequest) Validate() error {
	return validation.Fields(
		validation.Field("username", validation.Required(r.Username)),
		validation.Field("email", validation.Required(r.Email), validation.Email(r.Email)),
		validation.Field("password", validation.Required(r.Password), validation.Password(r.Password)),
	)
}

type LoginRequest struct {
	// Token represent either username or password for login attempt
	Token    string
	Password string
}

func (r LoginRequest) Validate() error {
	return validation.Fields(
		validation.Field("token", validation.Required(r.Token)),
		validation.Field("password", validation.Required(r.Password)),
	)
}

type serviceImpl struct {
	repository   principalRepository
	sessionStore sessionStore
	logger       *slog.Logger
}

// VerifySession implements [Service].
func (s *serviceImpl) VerifySession(ctx context.Context, sessionToken string) (*Principal, error) {
	principal, err := s.sessionStore.GetSession(ctx, sessionToken)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			return nil, apierrors.ErrUnauthorized
		}
		return nil, err
	}
	go func() {
		// context.WithoutCancel so the refresh outlives the request context.
		_, err := s.sessionStore.RefreshSession(context.WithoutCancel(ctx), sessionToken)
		if err != nil {
			s.logger.Error("Failed to refresh session", "sessionToken", sessionToken, "error", err)
		}
	}()
	return principal, nil
}

// Login implements [Service].
func (s *serviceImpl) Login(ctx context.Context, req LoginRequest) (string, error) {
	principal, err := s.repository.FindPrincipalByEmailOrUsername(ctx, req.Token)
	if err != nil {
		if errors.Is(err, errPrincipalNotFound) {
			return "", apierrors.ErrUnauthorized
		}
		return "", err
	}
	if !principal.VerifyPassword(req.Password) {
		return "", apierrors.ErrUnauthorized
	}
	s.logger.Debug("User logged in", "userId", principal.ID)
	sessionToken, err := s.sessionStore.CreateSession(ctx, principal.ID)
	if err != nil {
		return "", err
	}
	return sessionToken.Token, nil
}

// Logout implements [Service].
func (s *serviceImpl) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		s.logger.Debug("Logout called with empty session token")
		return nil
	}
	return s.sessionStore.DeleteSession(ctx, sessionToken)
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// Register implements [Service].
func (s *serviceImpl) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	principal, err := s.repository.CreatePrincipal(ctx, Principal{
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info("Created new user", "username", principal.Username, "email", principal.Email)
	session, err := s.sessionStore.CreateSession(ctx, principal.ID)
	if err != nil {
		return nil, err
	}
	return &RegisterResponse{
		Principal:    *principal,
		SessionToken: session.Token,
	}, err
}

var _ Service = (*serviceImpl)(nil)
