package user

import (
	"mrtutor-api/db/queries"
	"testing"
)

func TestCreateUserParamValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  CreateUserParams
		wantErr bool
	}{
		{
			name: "valid user",
			params: CreateUserParams{
				CreateUserParams: queries.CreateUserParams{
					Username: "validuser",
					Email:    "me@example.com",
					Password: "Validpassword00!",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			params: CreateUserParams{
				CreateUserParams: queries.CreateUserParams{
					Username: "validuser",
					Email:    "invalid-email",
					Password: "validpassword00",
				},
			},
			wantErr: true,
		},
		{
			name: "short password",
			params: CreateUserParams{
				CreateUserParams: queries.CreateUserParams{
					Username: "validuser",
					Email:    "me@example.com",
					Password: "short0",
				},
			},
			wantErr: true,
		},
		{
			name: "empty username",
			params: CreateUserParams{
				CreateUserParams: queries.CreateUserParams{
					Username: "   ",
					Email:    "me@example.com",
					Password: "validpassword00",
				},
			},
			wantErr: true,
		},
		{
			name: "empty email",
			params: CreateUserParams{
				CreateUserParams: queries.CreateUserParams{
					Username: "validuser",
					Email:    "   ",
					Password: "validpassword00",
				},
			},
			wantErr: true,
		},
		{
			name: "empty password",
			params: CreateUserParams{
				CreateUserParams: queries.CreateUserParams{
					Username: "validuser",
					Email:    "me@example.com",
					Password: "   ",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.params.validate(); (err != nil) != tt.wantErr {
				t.Errorf("User.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
