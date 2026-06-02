package validation

import (
	"regexp"
	"strings"
)

// IsValidEmail checks if the provided email is in a valid format
func IsValidEmail(email string) bool {
	// Simple regex for email validation
	const emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(emailRegex, email)
	return matched
}

var (
	passwordRegex  = regexp.MustCompile(`^[A-Za-z\d@#$%^&!?]{8,72}$`)
	lowercaseRegex = regexp.MustCompile(`[a-z]`)
	uppercaseRegex = regexp.MustCompile(`[A-Z]`)
	numberRegex    = regexp.MustCompile(`\d`)
	specialRegex   = regexp.MustCompile(`[@#$%^&!?]`)
)

// IsValidPassword checks if the password meets the criteria: between 8 and 72 characters long and
// contains at least one lowercase and uppercase letter, one number and at least one special character
func IsValidPassword(password string) string {
	if !passwordRegex.MatchString(password) {
		return "password must be between 8 and 72 characters long and contain only letters, numbers and special characters @#$%^&!?"
	}
	if !lowercaseRegex.MatchString(password) {
		return "password must contain at least one lowercase letter"
	}
	if !uppercaseRegex.MatchString(password) {
		return "password must contain at least one uppercase letter"
	}
	if !numberRegex.MatchString(password) {
		return "password must contain at least one number"
	}
	if !specialRegex.MatchString(password) {
		return "password must contain at least one special character (@#$%^&!?)"
	}
	return ""
}

// Required checks if a field is not empty after trimming whitespace
func Required(field string) bool {
	return strings.TrimSpace(field) != ""
}
