package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"
	"mrtutor-api/db/queries"
	"mrtutor-api/features/user"
	"net/http"
	"slices"
	"strconv"
	"testing"
)

func mustEncodeJSON(t *testing.T, v any) *bytes.Reader {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return bytes.NewReader(data)
}

func mustUnmarshalJSON[T any](t *testing.T, w io.Reader) T {
	result := new(T)
	decoder := json.NewDecoder(w)
	if err := decoder.Decode(result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	return *result
}

const defaultPassword = "Password123!"

func TestUserModule(t *testing.T) {
	db := setupTestDb(t)

	module := user.NewModule(db, slog.Default())
	mux := http.NewServeMux()
	module.RegisterRoutes(mux)

	client, url := setupTestClient(t, mux)

	t.Run("Can create user", func(t *testing.T) {
		t.Parallel()
		params := generateRandomCreateUserParams()
		body := mustEncodeJSON(t, params)
		resp, err := client.Post(url+"/users", "application/json", body)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Response body: %s", string(bodyBytes))
			return
		}
		created := mustUnmarshalJSON[user.User](t, resp.Body)
		if created.Email != params.Email {
			t.Errorf("Expected email %s, got %s", params.Email, created.Email)
		}
		if created.Username != params.Username {
			t.Errorf("Expected username %s, got %s", params.Username, created.Username)
		}
		if created.ID == 0 {
			t.Errorf("Expected non-zero user ID, got %d", created.ID)
		}
	})

	t.Run("Cannot create user with invalid email", func(t *testing.T) {
		t.Parallel()
		params := generateRandomCreateUserParams()
		params.Email = "me@example"
		body := mustEncodeJSON(t, params)
		resp, err := client.Post(url+"/users", "application/json", body)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status code 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Can login", func(t *testing.T) {
		t.Parallel()
		service := module.Service
		persisted, err := service.CreateUser(t.Context(), generateRandomCreateUserParams())
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		resp, err := client.Post(url+"/auth/login", "application/json", mustEncodeJSON(t, user.LoginUserParam{
			EmailOrUsername: persisted.Email,
			Password:        defaultPassword,
		}))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Response body: %s", string(bodyBytes))
			return
		}
		if !slices.ContainsFunc(resp.Cookies(), func(c *http.Cookie) bool {
			return c.Name == "session"
		}) {
			t.Errorf("Expected session cookie to be set")
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Logf("Response body: %s", string(bodyBytes))
			return
		}
	})
}

func generateRandomCreateUserParams() user.CreateUserParams {
	username := "me" + strconv.Itoa(rand.Int())
	return user.CreateUserParams{
		CreateUserParams: queries.CreateUserParams{
			Email:    username + "@example.com",
			Password: defaultPassword,
			Username: username,
		},
	}
}
