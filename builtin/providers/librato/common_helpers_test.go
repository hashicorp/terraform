package librato

import (
	"testing"
	"time"
)

func sleep(t *testing.T, amount time.Duration) func() {
	return func() {
		time.Sleep(amount * time.Second)
	}
}
