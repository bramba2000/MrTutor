package integration

import (
	"log/slog"
	"math/rand/v2"
	"mrtutor-api/features/auth"
	"net/http"
	"strconv"
	"testing"
)

func TestAuthModule(t *testing.T) {
	const defaultPassword = "Password123!"
	db := setupTestDb(t)

	module := auth.InitModule(db, slog.New(NewTextHandler(t)))
	mux := http.NewServeMux()
	module.RegisterRoutes(mux)

	client, url := setupTestClient(t, mux)

	generateRandomRegisterRequest := func() auth.RegisterRequest {
		username := "me" + strconv.Itoa(rand.Int())
		return auth.RegisterRequest{
			Username: username,
			Email:    username + "@example.com",
			Password: defaultPassword,
		}
	}

	t.Run("Can login", func(t *testing.T) {
		principal, err := module.Register(t.Context(), generateRandomRegisterRequest())
		if err != nil {
			t.Fatalf("Failed to register user: %v", err)
		}

		loginReq := auth.LoginRequest{
			Token:    principal.Username,
			Password: defaultPassword,
		}
		resp, err := client.Post(url+"/auth/login", "application/json", mustEncodeJSON(t, loginReq))

		if err != nil {
			t.Fatalf("Failed to send login request: %v", err)
		}
		defer resp.Body.Close() // nolint

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
			logResponseBody(t, resp)
		}

		var found bool
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "session" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected session cookie not found in response")
			t.Logf("%v", resp.Cookies())
		}
	})

	t.Run("Can register", func(t *testing.T) {
		principal := generateRandomRegisterRequest()

		registerRequest := auth.RegisterRequest{
			Email:    principal.Email,
			Username: principal.Username,
			Password: defaultPassword,
		}
		resp, err := client.Post(url+"/auth/register", "application/json", mustEncodeJSON(t, registerRequest))

		if err != nil {
			t.Fatalf("Failed to send login request: %v", err)
		}
		defer resp.Body.Close() // nolint

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
			logResponseBody(t, resp)
		}

		var found bool
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "session" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected session cookie not found in response")
			t.Logf("%v", resp.Cookies())
		}

		registerResponse := mustUnmarshalJSON[auth.Principal](t, resp.Body)
		if registerResponse.Username != principal.Username {
			t.Errorf("Expected username %s, got %s", principal.Username, registerResponse.Username)
		}
		if registerResponse.Email != principal.Email {
			t.Errorf("Expected email %s, got %s", principal.Email, registerResponse.Email)
		}
		if registerResponse.ID == 0 {
			t.Error("Expected non-zero user ID")
		}
	})

	t.Run("Can logout after login", func(t *testing.T) {
		principal, err := module.Register(t.Context(), generateRandomRegisterRequest())
		if err != nil {
			t.Fatalf("Failed to register user: %v", err)
		}

		req, err := http.NewRequest(http.MethodPost, url+"/auth/logout", http.NoBody)
		if err != nil {
			t.Fatalf("Failed to create logout request: %v", err)
		}
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: principal.SessionToken,
		})
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send logout request: %v", err)
		}
		defer resp.Body.Close() // nolint

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
			logResponseBody(t, resp)
		}
	})
}
