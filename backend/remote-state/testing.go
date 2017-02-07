package remotestate

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func TestClient(t *testing.T, raw backend.Backend) {
	b, ok := raw.(*Backend)
	if !ok {
		t.Fatalf("not Backend: %T", raw)
	}

	remote.TestClient(t, b.client)
}

// Test the lock implementation for a remote.Client.
// This test requires 2 backend instances, in oder to have multiple remote
// clients since some implementations may tie the client to the lock, or may
// have reentrant locks.
func TestRemoteLocks(t *testing.T, a, b backend.Backend) {
	sA, err := a.State()
	if err != nil {
		t.Fatal("failed to get state from backend A:", err)
	}

	sB, err := b.State()
	if err != nil {
		t.Fatal("failed to get state from backend B:", err)
	}

	lockerA, ok := sA.(state.Locker)
	if !ok {
		t.Fatal("client A not a state.Locker")
	}

	lockerB, ok := sB.(state.Locker)
	if !ok {
		t.Fatal("client B not a state.Locker")
	}

	if err := lockerA.Lock("test client A"); err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	if err := lockerB.Lock("test client B"); err == nil {
		lockerA.Unlock()
		t.Fatal("client B obtained lock while held by client A")
	} else {
		t.Log("lock info error:", err)
	}

	if err := lockerA.Unlock(); err != nil {
		t.Fatal("error unlocking client A", err)
	}

	if err := lockerB.Lock("test client B"); err != nil {
		t.Fatal("unable to obtain lock from client B")
	}

	if err := lockerB.Unlock(); err != nil {
		t.Fatal("error unlocking client B:", err)
	}

	// unlock should be repeatable
	if err := lockerA.Unlock(); err != nil {
		t.Fatal("Unlock error from client A when state was not locked:", err)
	}
}
