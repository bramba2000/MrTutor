package validation

import (
	"regexp"
	"strings"
)

type ValidationError struct {
	Problems []string
}

func (ve *ValidationError) Error() string {
	return "validation error: " + strings.Join(ve.Problems, "; ")
}

func NewValidationError(err error) *ValidationError {
	// If error implements Unwrap []error, we can unwrap it to get the underlying error message
	if unwrapped, ok := err.(interface{ Unwrap() []error }); ok {
		var problems []string
		for _, err := range unwrapped.Unwrap() {
			problems = append(problems, err.Error())
		}
		return &ValidationError{
			Problems: problems,
		}
	}
	return &ValidationError{
		Problems: []string{err.Error()},
	}
}

// IsValidEmail checks if the provided email is in a valid format
func IsValidEmail(email string) bool {
	// Simple regex for email validation
	const emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(emailRegex, email)
	return matched
}

// IsValidPassword checks if the password meets the criteria: between 8 and 72 characters long and
// contains at least one lowercase and uppercase letter, one number and at least one special character
func IsValidPassword(password string) bool {
	// Password must be at least 8 characters long and contain at least one letter and one number
	const passwordRegex = `^[A-Za-z\d@#$%^&!?]{8,}$`
	matched, err := regexp.MatchString(passwordRegex, password)
	if err != nil || !matched {
		return false
	}
	hasLowercase := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[@#$%^&!?]`).MatchString(password)
	return hasLowercase && hasNumber && hasSpecial && hasUppercase
}

// Required checks if a field is not empty after trimming whitespace
func Required(field string) bool {
	return strings.TrimSpace(field) != ""
}
