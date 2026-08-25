// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"testing"
	"time"
)

func TestNewHandleWithDependency_NoParentDoesNotLeakLock(t *testing.T) {

	// similar to serverHandshake in plugin.go
	tbl := newHandleTable()

	// Tryong to reproduce a race scenario (eg: closeSourceBundle completes
	// concurrently with an inflight OpenStackConfiguration)
	depHnd := newHandle(tbl, "dependency-obj")
	if err := closeHandle(tbl, depHnd); err != nil {
		t.Fatalf("close handle errors: %v", err)
	}

	_, err := newHandleWithDependency(tbl, "obj", depHnd)
	if err != newHandleErrorNoParent {
		t.Fatalf("expected no parent error, got %v", err)
	}
	// when no parent error is returned newHandleWithDependency does not release the lock. so calls after would block.
	// adding a timeout here so as not to break ci. 2secs only.
	done := make(chan handle[string], 1)
	go func() {
		hnd := newHandle(tbl, "another-obj")
		done <- hnd
	}()

	select {
	case <-done:
		// mutex was released correctly
	case <-time.After(2 * time.Second):
		t.Fatal("handleTable t.mu was not released. deadlocked.")
	}
}
