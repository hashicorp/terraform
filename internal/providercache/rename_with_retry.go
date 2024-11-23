package providercache

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// renameWithRetry attempts to rename a file, retrying on Windows for transient errors.
func renameWithRetry(from, to string) error {
	// On non-Windows systems, perform a simple rename.
	if runtime.GOOS != "windows" {
		return os.Rename(from, to)
	}

	// Backoff configuration.
	const (
		initialInterval = 10 * time.Millisecond
		maxElapsedTime  = 2 * time.Second
		maxRetries      = 5
	)

	interval := initialInterval
	elapsedTime := 0 * time.Millisecond

	for attempts := 0; attempts < maxRetries && elapsedTime < maxElapsedTime; attempts++ {
		err := os.Rename(from, to)
		if err == nil {
			return nil // Success
		}

		if !isTransientError(err) {
			return err // Permanent error
		}

		// Log and backoff before retrying.
		fmt.Printf("[WARN] Retrying rename from %s to %s due to transient error: %v\n", from, to, err)
		sleepTime := time.Duration(rand.Int63n(int64(interval)))
		time.Sleep(sleepTime)
		elapsedTime += sleepTime
		interval *= 2 // Exponential backoff
	}

	return fmt.Errorf("failed to rename %s to %s after retries", from, to)
}

// isTransientError determines if an error is transient, focusing on Windows-specific scenarios.
func isTransientError(err error) bool {
	if errors.Is(err, os.ErrPermission) {
		// On Windows, non-atomic file renames might cause transient permission errors.
		return true
	}
	return false
}
