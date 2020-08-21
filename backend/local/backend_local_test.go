package local

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/internal/initwd"
)

func TestLocalContext(t *testing.T) {
	configDir := "./testdata/empty"
	b, cleanup := TestLocal(t)
	defer cleanup()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
	defer configCleanup()

	op := &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		Workspace:    backend.DefaultStateName,
		LockState:    true,
	}

	_, _, diags := b.Context(op)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err().Error())
	}

	// Context() retains a lock on success
	assertBackendStateLocked(t, b)
}

func TestLocalContext_error(t *testing.T) {
	configDir := "./testdata/apply"
	b, cleanup := TestLocal(t)
	defer cleanup()

	_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
	defer configCleanup()

	op := &backend.Operation{
		ConfigDir:    configDir,
		ConfigLoader: configLoader,
		Workspace:    backend.DefaultStateName,
		LockState:    true,
	}

	_, _, diags := b.Context(op)
	if !diags.HasErrors() {
		t.Fatal("unexpected success")
	}

	// Context() unlocks the state on failure
	assertBackendStateUnlocked(t, b)

}
