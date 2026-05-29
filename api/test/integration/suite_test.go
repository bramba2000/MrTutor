package integration

import (
	"database/sql"
	"flag"
	"io"
	"mrtutor-api/db"
	"mrtutor-api/db/migrations"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var integration = flag.Bool("it", false, "enable running of integration testing")

func TestMain(m *testing.M) {
	flag.Parse()
	if !*integration {
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func setupTestDb(t *testing.T) *sql.DB {
	t.Helper()
	db, err := db.NewInMemory()
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	m, err := migrations.NewWithDb(db)
	if err != nil {
		t.Fatalf("failed to create migrations: %v", err)
	}
	t.Cleanup(func() {
		m.Close()
	})
	if err := m.Up(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	return db
}

func setupTestClient(t *testing.T, mux *http.ServeMux) (*http.Client, string) {
	t.Helper()
	testServer := httptest.NewServer(mux)
	t.Cleanup(func() {
		testServer.Close()
	})
	return testServer.Client(), testServer.URL
}

func logResponseBody(t *testing.T, resp *http.Response) {
	t.Helper()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Logf("failed to read response body: %v", err)
		return
	}
	t.Logf("response body: %s", string(bodyBytes))
}
