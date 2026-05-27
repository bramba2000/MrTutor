package e2e

import (
	"fmt"
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
