package e2e

import (
	"fmt"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func checkHealthy(t *testing.T, host string) error {
	resp, err := doRequest(t, host, http.MethodGet, "/health", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to make health request: %w", err)
	}
	defer resp.Body.Close() //nolint
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func TestGracefulShutdown(t *testing.T) {
	t.Parallel()
	host, cmd := startApplication(t)

	err := Retry(func() error {
		return checkHealthy(t, host)
	}, 5, 500*time.Millisecond, 3*time.Second)
	if err != nil {
		t.Fatalf("Application did not start successfully: %v", err)
	}

	err = cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send SIGTERM signal: %v", err)
	}

	err = Retry(func() error {
		if err := checkHealthy(t, host); err == nil {
			return fmt.Errorf("application is still healthy, expected to be shutting down")
		}
		return nil
	}, 5, 500*time.Millisecond, 3*time.Second)

	state, err := cmd.Process.Wait()
	if err != nil {
		t.Fatalf("Failed to wait for application process: %v", err)
	}
	if state.ExitCode() != 0 {
		t.Fatalf("Application did not exit gracefully, exit code: %d", state.ExitCode())
	}
}
