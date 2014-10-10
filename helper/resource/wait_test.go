package resource

import (
	"fmt"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	t.Parallel()

	tries := 0
	f := func() error {
		tries++
		if tries == 1 {
			return nil
		}

		return fmt.Errorf("error")
	}

	err := Retry(2*time.Second, f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestRetry_timeout(t *testing.T) {
	t.Parallel()

	f := func() error {
		return fmt.Errorf("always")
	}

	err := Retry(1*time.Second, f)
	if err == nil {
		t.Fatal("should error")
	}
}
