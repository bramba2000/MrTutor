package user

import (
	"errors"
	"mrtutor-api/db/queries"
	. "mrtutor-api/validation"
)

var (
	ErrEmailRequired    = errors.New("email is required")
	ErrEmailInvalid     = errors.New("email is invalid")
	ErrPasswordRequired = errors.New("password is required")
	ErrNameRequired     = errors.New("name is required")
)

type CreateUserParams struct {
	queries.CreateUserParams
}

func (p CreateUserParams) validate() error {
	var errorList []error

	if !Required(p.Email) {
		errorList = append(errorList, ErrEmailRequired)
	} else if !IsValidEmail(p.Email) {
		errorList = append(errorList, ErrEmailInvalid)
	}

	if !Required(p.Password) {
		errorList = append(errorList, ErrPasswordRequired)
	}

	if !Required(p.Username) {
		errorList = append(errorList, ErrNameRequired)
	}

	return errors.Join(errorList...)
}

func (p CreateUserParams) validatePassword() (err *ValidationError) {
	if msg := IsValidPassword(p.Password); msg != "" {
		return &ValidationError{Problems: []string{msg}}
	}
	return nil
}

type User queries.User
