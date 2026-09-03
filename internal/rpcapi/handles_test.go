// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"testing"
	"time"

	"github.com/hashicorp/go-slug/sourcebundle"

	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
)

func TestNewHandleWithDependency_ParentNotExists(t *testing.T) {
	// This test verifies that when newHandleWithDependency returns
	// newHandleErrorNoParent, the mutex is properly unlocked and does
	// not leak. This was a bug where the mutex was not unlocked on
	// the error path, causing a potential deadlock.
	//
	// See: https://github.com/hashicorp/terraform/issues/39053

	tbl := newHandleTable()

	// Create a handle that depends on a non-existent parent handle (id 999)
	nonExistentParent := handle[*sourcebundle.Bundle](999)
	_, err := newHandleWithDependency(tbl, &stackconfig.Config{}, nonExistentParent)

	if err != newHandleErrorNoParent {
		t.Fatalf("expected newHandleErrorNoParent, got %v", err)
	}

	// Now verify the mutex is not leaked by attempting to acquire it.
	// If the mutex was leaked, this would block forever (or timeout).
	// We use a goroutine with a timeout to detect the leak.
	done := make(chan struct{})
	go func() {
		tbl.mu.Lock()
		tbl.mu.Unlock()
		close(done)
	}()

	select {
	case <-done:
		// Mutex was properly unlocked - test passes
	case <-time.After(time.Second):
		t.Fatal("mutex appears to be leaked: could not acquire lock within timeout")
	}
}