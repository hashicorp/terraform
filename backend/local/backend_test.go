package local

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocal_impl(t *testing.T) {
	var _ backend.Enhanced = new(Local)
	var _ backend.Local = new(Local)
}

func checkState(t *testing.T, path, expected string) {
	// Read the state
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected = strings.TrimSpace(expected)
	if actual != expected {
		t.Fatalf("state does not match! actual:\n%s\n\nexpected:\n%s", actual, expected)
	}
}
