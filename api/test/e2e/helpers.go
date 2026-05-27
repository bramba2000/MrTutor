package e2e

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func Retry(fn func() error, maxRetry int, startBackoff, maxBackoff time.Duration) error {
	for attempt := 0; ; attempt++ {
		if err := fn(); err == nil {
			return err
		} else if attempt == maxRetry-1 {
			return fmt.Errorf("failed after %d attempts: %w", maxRetry, err)
		}

		time.Sleep(startBackoff)
		if startBackoff < maxBackoff {
			startBackoff *= 2
		}
	}
}

func doRequest(t testing.TB, host, method, path string, body io.Reader) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, host+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return http.DefaultClient.Do(req)
}

// freePort returns a free TCP port on loopback by briefly binding :0 and releasing it.
func freePort(t testing.TB) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to obtain free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return strconv.Itoa(port)
}
