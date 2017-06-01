package librato

import (
	"log"
	"testing"
	"time"
)

func sleep(t *testing.T, amount time.Duration) func() {
	return func() {
		log.Printf("[INFO] Sleeping for %d seconds...", amount)
		time.Sleep(amount * time.Second)
	}
}
