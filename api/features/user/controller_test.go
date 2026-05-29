package user

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	apierrors "mrtutor-api/errors"
	"mrtutor-api/validation"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserRoutes(t *testing.T) {
	const existingEmail = "existing@example.com"
	const password = "password"

	service := mockUserService{
		CreateUserFunc: func(ctx context.Context, p CreateUserParams) (User, error) {
			switch p.Username {
			case "valid":
				return User{ID: 1}, nil
			case "invalid":
				return User{}, &validation.ValidationError{
					Problems: []string{"invalid params"},
				}
			case "internalError":
				return User{}, errors.New("internal error")
			default:
				panic("unexpected user id" + p.Username)
			}
		},
		LoginUserFunc: func(ctx context.Context, p LoginUserParam) (string, error) {
			switch p.EmailOrUsername {
			case "valid":
				return "token", nil
			case "wrong":
				return "", apierrors.ErrUnauthorized
			default:
				panic("unexpected user id" + p.EmailOrUsername)
			}
		},
	}

	mux := http.NewServeMux()

	controller := UserController{
		service: service,
	}
	controller.RegisterRoutes(mux)
	tServer := httptest.NewServer(mux)
	defer tServer.Close()
	tClient := tServer.Client()

	t.Run("Create user", func(t *testing.T) {
		tt := []struct {
			name         string
			username     string
			expectStatus int
		}{
			{
				name:         "valid",
				username:     "valid",
				expectStatus: http.StatusCreated,
			}, {
				name:         "invalid",
				username:     "invalid",
				expectStatus: http.StatusBadRequest,
			}, {
				name:         "internal error",
				username:     "internalError",
				expectStatus: http.StatusInternalServerError,
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				params := newTestCreateUserParams(tc.username)
				body := new(bytes.Buffer)
				if err := json.NewEncoder(body).Encode(params); err != nil {
					t.Fatalf("Failed to encode request body: %v", err)
				}
				req, _ := http.NewRequest("POST", tServer.URL+"/users", body)
				resp, err := tClient.Do(req)
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tc.expectStatus {
					t.Errorf("Expected status %d, got %d", tc.expectStatus, resp.StatusCode)
				}
			})
		}

	})

	t.Run("Login", func(t *testing.T) {
		tt := []struct {
			name         string
			username     string
			expectStatus int
		}{
			{
				name:         "Valid",
				username:     "valid",
				expectStatus: http.StatusOK,
			}, {
				name:         "Wrong creadentials",
				username:     "wrong",
				expectStatus: http.StatusUnauthorized,
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				params := LoginUserParam{
					EmailOrUsername: tc.username,
					Password:        password,
				}
				body := new(bytes.Buffer)
				if err := json.NewEncoder(body).Encode(params); err != nil {
					t.Fatalf("Failed to encode request body: %v", err)
				}
				req, _ := http.NewRequest("POST", tServer.URL+"/auth/login", body)
				resp, err := tClient.Do(req)
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tc.expectStatus {
					t.Errorf("Expected status %d, got %d", tc.expectStatus, resp.StatusCode)
				}
			})
		}
	})
}

func newTestCreateUserParams(username string) CreateUserParams {
	params := CreateUserParams{}
	params.Email = "me@example.com"
	params.Username = cmp.Or(username, "me")
	params.Password = "password"
	return params
}

type mockUserService struct {
	CreateUserFunc func(ctx context.Context, params CreateUserParams) (User, error)
	LoginUserFunc  func(ctx context.Context, params LoginUserParam) (string, error)
}

// CreateUser implements [UserService].
func (m mockUserService) CreateUser(ctx context.Context, params CreateUserParams) (User, error) {
	return m.CreateUserFunc(ctx, params)
}

// LoginUser implements [UserService].
func (m mockUserService) LoginUser(ctx context.Context, params LoginUserParam) (string, error) {
	return m.LoginUserFunc(ctx, params)
}

var _ UserService = mockUserService{}
