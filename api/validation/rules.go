package validation

import (
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"strings"
)

const (
	maxPasswordLength = 72
	minPasswordLength = 8
)

var (
	ErrRequired           = errors.New("must be provided")
	ErrPasswordLength     = fmt.Errorf("password must be between %d and %d characters", minPasswordLength, maxPasswordLength)
	ErrPasswordComplexity = errors.New("password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character")
)

// Required checks if the value is not nil, empty string, empty slice, or empty map.
func Required(value any) error {
	if value == nil {
		return ErrRequired
	}

	// Use reflection to handle all slice and map underlying types
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		if strings.TrimSpace(v.String()) == "" {
			return ErrRequired
		}
		return nil

	case reflect.Slice, reflect.Map:
		if v.Len() == 0 {
			return ErrRequired
		}
		return nil

	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return ErrRequired
		}
		// Recursively evaluate the element the pointer points to
		return Required(v.Elem().Interface())

	default:
		// For scalars (int, bool, structs) that are not nil, they are considered "provided"
		return nil
	}
}

// Email checks if the value is a valid email address.
func Email(value string) error {
	_, err := mail.ParseAddress(value)
	return err
}

// Password checks if the value satisfies the password requirements
//
// A password must have between 8 and 72 characters, at least one
// uppercase letter, one lowercase letter, one digit, and one special character.
func Password(value string) error {
	if len(value) < minPasswordLength || len(value) > maxPasswordLength {
		return ErrPasswordLength
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range value {
		switch {
		case 'A' <= c && c <= 'Z':
			hasUpper = true
		case 'a' <= c && c <= 'z':
			hasLower = true
		case '0' <= c && c <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()-_=+[]{}|;:',.<>?/", c):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return ErrPasswordComplexity
	}

	return nil
}
