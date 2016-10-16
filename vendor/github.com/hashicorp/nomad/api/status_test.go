package api

import (
	"testing"
)

func TestStatus_Leader(t *testing.T) {
	c, s := makeClient(t, nil, nil)
	defer s.Stop()
	status := c.Status()

	// Query for leader status should return a result
	out, err := status.Leader()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if out == "" {
		t.Fatalf("expected leader, got: %q", out)
	}
}
