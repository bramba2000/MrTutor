package errors

import "errors"

type NotFoundError struct {
	Entity string
}

func (e NotFoundError) Error() string {
	return e.Entity + " not found"
}

var (
	// ErrUnauthorized indicates the request is not authenticated (maps to 401).
	ErrUnauthorized = errors.New("Unauthorized")
	// ErrForbidden indicates the principal is authenticated but not allowed to
	// perform the action (maps to 403). Return it from authorization checks.
	ErrForbidden = errors.New("Forbidden")
)
