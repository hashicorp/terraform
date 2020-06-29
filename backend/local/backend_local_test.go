package local

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/states/statemgr"
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

	// Conext() retains a lock on success, so this should fail.
	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err == nil {
		t.Fatalf("unexpected success locking state")
	}
}

func TestLocalContext_error(t *testing.T) {
	configDir := "./testdata/apply-error"
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

	// When Context() returns an error, it also unlocks the state.
	// This should therefore succeed.
	stateMgr, _ := b.StateMgr(backend.DefaultStateName)
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state: %s", err.Error())
	}
}
