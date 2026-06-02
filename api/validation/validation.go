package validation

import (
	"strings"
)

type Error struct {
	Problems []string
}

func (ve *Error) Error() string {
	return "validation error: " + strings.Join(ve.Problems, "; ")
}

func NewValidationError(err error) *Error {
	// If error implements Unwrap []error, we can unwrap it to get the underlying error message
	if unwrapped, ok := err.(interface{ Unwrap() []error }); ok {
		var problems []string
		for _, err := range unwrapped.Unwrap() {
			problems = append(problems, err.Error())
		}
		return &Error{
			Problems: problems,
		}
	}
	return &Error{
		Problems: []string{err.Error()},
	}
}

type Validable interface {
	Validate() error
}
