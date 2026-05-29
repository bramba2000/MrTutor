package errors

import "errors"

type NotFoundError struct {
	Entity string
}

func (e NotFoundError) Error() string {
	return e.Entity + " not found"
}

var (
	ErrUnauthorized = errors.New("Unauthorized")
)
