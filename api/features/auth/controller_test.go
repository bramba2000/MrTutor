package auth

import (
	"bytes"
	"context"
	"encoding/json"
	apierrors "mrtutor-api/errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:generate go tool moq -out controller_moq_test.go . Service

func TestController(t *testing.T) {
	password := "pswd"
	hashedPassword := "hashed_pswd"
	principal := Principal{
		ID:             1,
		Username:       "me",
		Email:          "me@example.com",
		HashedPassword: hashedPassword,
	}

	const validSessionToken = "sessionToken"
	svc := &ServiceMock{
		LoginFunc: func(ctx context.Context, req LoginRequest) (string, error) {
			if req.Token != principal.Username {
				return "", apierrors.ErrUnauthorized
			} else if req.Password != password {
				return "", apierrors.ErrUnauthorized
			}
			return validSessionToken, nil
		},
		LogoutFunc: func(ctx context.Context, sessionToken string) error {
			switch sessionToken {
			case validSessionToken:
			case "":
				return nil
			default:
				return errSessionNotFound
			}
			return nil
		},
	}
	controller := controller{
		service: svc,
	}

	t.Run("Status 401 when no credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		w := httptest.NewRecorder()
		controller.LoginHandler().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("Status 401 when invalid password", func(t *testing.T) {
		body, _ := json.Marshal(LoginRequest{
			Token:    principal.Username,
			Password: "invalid",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		controller.LoginHandler().ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("Status 401 when invalid token", func(t *testing.T) {
		body, _ := json.Marshal(LoginRequest{
			Token:    "invalid",
			Password: password,
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		controller.LoginHandler().ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("Status 200 when logout without session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		w := httptest.NewRecorder()
		controller.LogoutHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
			t.Logf("response body: %s", w.Body.String())
		}
		var found bool
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == sessionCookieName && cookie.Value == "" {
				found = true
				break
			}
		}
		if found {
			t.Errorf("expected session cookie to be cleared")
		}
	})

	t.Run("Status 200 when logout", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: validSessionToken})
		w := httptest.NewRecorder()
		controller.LogoutHandler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
			t.Logf("response body: %s", w.Body.String())
		}
		var found bool
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == sessionCookieName && cookie.Value == "" {
				found = true
				break
			}
		}
		if found {
			t.Errorf("expected session cookie to be cleared")
		}
	})
}
