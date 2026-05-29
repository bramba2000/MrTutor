package user

import (
	"context"
	"fmt"
	"log/slog"
	apierrors "mrtutor-api/errors"

	"golang.org/x/crypto/bcrypt"
)

type LoginUserParam struct {
	EmailOrUsername string `json:"login"`
	Password        string `json:"password"`
}

type UserService interface {
	// CreateUser creates a new user with the given parameters.
	CreateUser(ctx context.Context, params CreateUserParams) (User, error)
	// LoginUser authenticates a user with the given credentials and returns a token if successful.
	LoginUser(ctx context.Context, params LoginUserParam) (string, error)
}

type userService struct {
	repo           UserRepository
	logger         *slog.Logger
	sessionManager SessionManager
}

// LoginUser implements [UserService].
func (u userService) LoginUser(ctx context.Context, params LoginUserParam) (string, error) {
	user, err := u.repo.GetUserByEmailOrUsername(ctx, params.EmailOrUsername)
	if err != nil {
		if err == ErrUserNotFound {
			return "", apierrors.ErrUnauthorized
		}
		return "", err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
	if err != nil {
		return "", apierrors.ErrUnauthorized
	}
	session, err := u.sessionManager.CreateSession(ctx, user.ID)
	if err != nil {
		return "", err
	}
	u.logger.Debug(fmt.Sprintf("user %d successfully login", user.ID))
	return session.Token, nil
}

// CreateUser implements [UserService].
func (u userService) CreateUser(ctx context.Context, params CreateUserParams) (User, error) {
	if err := params.validatePassword(); err != nil {
		return User{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(params.Password), 14)
	if err != nil {
		return User{}, err
	}
	params.Password = string(passwordHash)
	return u.repo.CreateUser(ctx, params)
}

func NewUserService(repo UserRepository, sessionManager SessionManager, logger *slog.Logger) UserService {
	return userService{repo, logger, sessionManager}
}
