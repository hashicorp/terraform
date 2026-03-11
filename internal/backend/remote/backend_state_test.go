// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package remote

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(remoteClient)
}

func TestRemoteClient(t *testing.T) {
	client := testRemoteClient(t)
	remote.TestClient(t, client)
}

func TestRemoteClient_stateLock(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	s1, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("expected no error, got %v", sDiags.Err())
	}

	s2, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("expected no error, got %v", sDiags.Err())
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}

func TestRemoteClient_Unlock_invalidID(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	s1, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("expected no error, got %v", sDiags.Err())
	}

	err := s1.Unlock("no")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "does not match existing lock ID") {
		t.Fatalf("expected erroor containing \"does not match existing lock ID\", got %v", err)
	}
}

func TestRemoteClient_Unlock(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	s1, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatalf("expected no error, got %v", sDiags.Err())
	}

	id, err := s1.Lock(&statemgr.LockInfo{
		ID: "test",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = s1.Unlock(id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRemoteClient_Put_withRunID(t *testing.T) {
	// Set the TFE_RUN_ID environment variable before creating the client!
	if err := os.Setenv("TFE_RUN_ID", cloud.GenerateID("run-")); err != nil {
		t.Fatalf("error setting env var TFE_RUN_ID: %v", err)
	}

	// Create a new test client.
	client := testRemoteClient(t)

	// Create a new empty state.
	sf := statefile.New(states.NewState(), "", 0)
	var buf bytes.Buffer
	statefile.Write(sf, &buf)

	// Store the new state to verify (this will be done
	// by the mock that is used) that the run ID is set.
	if diags := client.Put(buf.Bytes()); diags.HasErrors() {
		t.Fatalf("expected no error, got %v", diags.Err())
	}
}
