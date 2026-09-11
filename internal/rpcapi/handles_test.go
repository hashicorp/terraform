// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"testing"
)

// assertTableUnlocked fails the test if the handle table's mutex is still held.
//
// TryLock is used rather than racing a goroutine against a timeout so that the
// assertion is deterministic: a held mutex fails immediately instead of after a
// wait, and a released one costs nothing.
func assertTableUnlocked(t *testing.T, tbl *handleTable) {
	t.Helper()
	if !tbl.mu.TryLock() {
		t.Fatal("handleTable.mu is still held; subsequent handle operations would block forever")
	}
	tbl.mu.Unlock()
}

func TestNewHandleWithDependencyNoParentReleasesLock(t *testing.T) {
	tbl := newHandleTable()

	// Close the dependency first, so that creating a handle against it takes
	// the same path as an OpenStackConfiguration racing a CloseSourceBundle.
	depHnd := newHandle(tbl, "dependency-obj")
	if err := closeHandle(tbl, depHnd); err != nil {
		t.Fatalf("closing the dependency handle: %s", err)
	}

	if _, err := newHandleWithDependency(tbl, "obj", depHnd); err != newHandleErrorNoParent {
		t.Fatalf("wrong error for a missing parent\ngot:  %v\nwant: %v", err, newHandleErrorNoParent)
	}

	assertTableUnlocked(t, tbl)
}

func TestNewHandleWithDependencyWrongTypeReleasesLock(t *testing.T) {
	tbl := newHandleTable()

	// The dependency is alive but holds a string, while the lookup below asks
	// for an int, so the type assertion panics. That panic is the caller's
	// fault and is meant to be fatal, but it must not leave the table locked:
	// anything recovering from it would otherwise deadlock every later call.
	depHnd := newHandle(tbl, "dependency-obj")
	wrongType := handle[int](depHnd)

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected a panic for a dependency handle of the wrong type")
			}
		}()
		newHandleWithDependency(tbl, "obj", wrongType)
	}()

	assertTableUnlocked(t, tbl)
}

func TestNewHandleWithDependencySuccessReleasesLock(t *testing.T) {
	tbl := newHandleTable()

	depHnd := newHandle(tbl, "dependency-obj")
	hnd, err := newHandleWithDependency(tbl, "obj", depHnd)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got, ok := readHandle(tbl, hnd); !ok || got != "obj" {
		t.Fatalf("new handle did not read back\ngot:  %q, %t\nwant: %q, true", got, ok, "obj")
	}

	assertTableUnlocked(t, tbl)
}
