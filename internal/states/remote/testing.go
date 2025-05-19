// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"bytes"
	"testing"

	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

// TestClient is a generic function to test any client.
func TestClient(t *testing.T, c Client) {
	var buf bytes.Buffer
	s := statemgr.TestFullInitialState()
	sf := statefile.New(s, "stub-lineage", 2)
	err := statefile.Write(sf, &buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	data := buf.Bytes()

	if err := c.Put(data); err != nil {
		t.Fatalf("put: %s", err)
	}

	p, err := c.Get()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if !bytes.Equal(p.Data, data) {
		t.Fatalf("expected full state %q\n\ngot: %q", string(p.Data), string(data))
	}

	if err := c.Delete(); err != nil {
		t.Fatalf("delete: %s", err)
	}

	p, err = c.Get()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if p != nil {
		t.Fatalf("expected empty state, got: %q", string(p.Data))
	}
}

// Test the lock implementation for a remote.Client.
// This test requires 2 client instances, in oder to have multiple remote
// clients since some implementations may tie the client to the lock, or may
// have reentrant locks.
func TestRemoteLocks(t *testing.T, a, b Client) {
	lockerA, ok := a.(statemgr.Locker)
	if !ok {
		t.Fatal("client A not a statemgr.Locker")
	}

	lockerB, ok := b.(statemgr.Locker)
	if !ok {
		t.Fatal("client B not a statemgr.Locker")
	}

	infoA := statemgr.NewLockInfo()
	infoA.Operation = "test"
	infoA.Who = "clientA"

	infoB := statemgr.NewLockInfo()
	infoB.Operation = "test"
	infoB.Who = "clientB"

	lockIDA, err := lockerA.Lock(infoA)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	_, err = lockerB.Lock(infoB)
	if err == nil {
		lockerA.Unlock(lockIDA)
		t.Fatal("client B obtained lock while held by client A")
	}
	if _, ok := err.(*statemgr.LockError); !ok {
		t.Errorf("expected a LockError, but was %t: %s", err, err)
	}

	if err := lockerA.Unlock(lockIDA); err != nil {
		t.Fatal("error unlocking client A", err)
	}

	lockIDB, err := lockerB.Lock(infoB)
	if err != nil {
		t.Fatal("unable to obtain lock from client B")
	}

	if lockIDB == lockIDA {
		t.Fatalf("duplicate lock IDs: %q", lockIDB)
	}

	if err = lockerB.Unlock(lockIDB); err != nil {
		t.Fatal("error unlocking client B:", err)
	}

	// TODO: Should we enforce that Unlock requires the correct ID?
}
