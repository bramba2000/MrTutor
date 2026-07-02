package validation_test

import (
	"mrtutor/api/validation"
	"strings"
	"testing"
)

func TestEmail(t *testing.T) {
	// Test cases for email validation
	testCases := []struct {
		name          string
		input         string
		expectedError bool
	}{
		{"Valid email", "me@example.com", false},
		{"Invalid email - missing @", "meexample.com", true},
		{"Invalid email - missing domain", "me@", true},
		{"Invalid email - missing username", "@example.com", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validation.Email(tc.input)
			if (result != nil) != tc.expectedError {
				t.Errorf("Email(%q) = %v; want error: %v", tc.input, result, tc.expectedError)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	// Test cases for required field validation
	testCases := []struct {
		name          string
		input         any
		expectedError bool
	}{
		{"Non-empty string", "Hello", false},
		{"Empty string", "", true},
		{"Space only string", "   ", true},
		{"Non-empty slice", []int{1, 2, 3}, false},
		{"Empty slice", []int{}, true},
		{"Non-empty map", map[string]int{"key": 1}, false},
		{"Empty map", map[string]int{}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validation.Required(tc.input)
			if (result != nil) != tc.expectedError {
				t.Errorf("Required(%v) = %v; want error: %v", tc.input, result, tc.expectedError)
			}
		})
	}
}

func TestPassword(t *testing.T) {
	// Test cases for password validation
	testCases := []struct {
		name          string
		input         string
		expectedError bool
	}{
		{"Valid password", "Password123!", false},
		{"Invalid password - too short", "Pass1!", true},
		{"Invalid password - too long", strings.Repeat("Password123!", 10), true},
		{"Invalid password - no uppercase", "password123!", true},
		{"Invalid password - no lowercase", "PASSWORD123!", true},
		{"Invalid password - no digit", "Password!", true},
		{"Invalid password - no special character", "Password123", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validation.Password(tc.input)
			if (result != nil) != tc.expectedError {
				t.Errorf("Password(%q) = %v; want error: %v", tc.input, result, tc.expectedError)
			}
		})
	}
}
