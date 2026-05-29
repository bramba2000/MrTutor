package user

import (
	"context"
	"database/sql"
	"mrtutor-api/db/queries"
	apierrors "mrtutor-api/errors"
	. "mrtutor-api/validation"
)

var (
	ErrUserNotFound = apierrors.NotFoundError{Entity: "user"}
)

type UserRepository interface {
	// CreateUser creates a new user in the database. Return [apierrors.ValidationError]
	// if params generate invalid user
	CreateUser(ctx context.Context, param CreateUserParams) (User, error)
	GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (User, error)
}

type userRepository struct {
	queries *queries.Queries
}

// GetUserByEmailOrUsername implements [UserRepository].
func (u userRepository) GetUserByEmailOrUsername(ctx context.Context, emailOrUsername string) (User, error) {
	if user, err := u.queries.GetUserByEmailOrUsername(ctx, emailOrUsername); err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	} else {
		return User(user), nil
	}
}

// CreateUser implements [UserRepository].
func (u userRepository) CreateUser(ctx context.Context, param CreateUserParams) (User, error) {
	if err := param.validate(); err != nil {
		return User{}, NewValidationError(err)
	}
	if user, err := u.queries.CreateUser(ctx, param.CreateUserParams); err != nil {
		return User{}, err
	} else {
		return User(user), nil
	}
}

func newSQLUserRepository(db *sql.DB) UserRepository {
	return userRepository{queries: queries.New(db)}
}
