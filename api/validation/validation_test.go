package validation_test

import (
	"mrtutor-api/validation"
	"testing"
)

func TestValidPassword(t *testing.T) {
	tt := []struct {
		name     string
		password string
		valid    bool
	}{
		{
			name:     "valid password",
			password: "P@ssw0rd",
			valid:    true,
		},
		{
			name:     "too short",
			password: "P@ss1",
			valid:    false,
		},
		{
			name:     "no uppercase",
			password: "p@ssw0rd",
			valid:    false,
		},
		{
			name:     "no lowercase",
			password: "P@SSW0RD",
			valid:    false,
		},
		{
			name:     "no digit",
			password: "P@ssword",
			valid:    false,
		},
		{
			name:     "no special character",
			password: "Passw0rd",
			valid:    false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			valid := validation.IsValidPassword(tc.password)
			if valid != tc.valid {
				t.Errorf("validatePassword(%q) got = %v, wantErr %v", tc.password, valid, tc.valid)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tt := []struct {
		name  string
		value string
		valid bool
	}{
		{
			name:  "non-empty string",
			value: "hello",
			valid: true,
		},
		{
			name:  "empty string",
			value: "",
			valid: false,
		},
		{
			name:  "space only string",
			value: "  ",
			valid: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			valid := validation.Required(tc.value)
			if valid != tc.valid {
				t.Errorf("Required(%q) got = %v, wantErr %v", tc.value, valid, tc.valid)
			}
		})
	}
}

func TestValidEmail(t *testing.T) {
	tt := []struct {
		name  string
		email string
		valid bool
	}{
		{
			name:  "valid email",
			email: "me@example.com",
			valid: true,
		},
		{
			name:  "missing @",
			email: "meexample.com",
			valid: false,
		},
		{
			name:  "missing domain",
			email: "me@",
			valid: false,
		},
		{
			name:  "missing username",
			email: "@example.com",
			valid: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			valid := validation.IsValidEmail(tc.email)
			if valid != tc.valid {
				t.Errorf("IsValidEmail(%q) got = %v, wantErr %v", tc.email, valid, tc.valid)
			}
		})
	}
}
