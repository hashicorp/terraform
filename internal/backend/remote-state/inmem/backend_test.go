// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmem

import (
	"flag"
	"os"
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/backend"
	statespkg "github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	defer Reset()
	testID := "test_lock_id"

	config := map[string]interface{}{
		"lock_id": testID,
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	c := s.(*remote.State).Client.(*RemoteClient)
	if c.Name != backend.DefaultStateName {
		t.Fatal("client name is not configured")
	}

	if err := locks.unlock(backend.DefaultStateName, testID); err != nil {
		t.Fatalf("default state should have been locked: %s", err)
	}
}

func TestBackend(t *testing.T) {
	defer Reset()
	b := backend.TestBackendConfig(t, New(), hcl.EmptyBody()).(*Backend)
	backend.TestBackendStates(t, b)
}

func TestBackendLocked(t *testing.T) {
	defer Reset()
	b1 := backend.TestBackendConfig(t, New(), hcl.EmptyBody()).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), hcl.EmptyBody()).(*Backend)

	backend.TestBackendStateLocks(t, b1, b2)
}

// use the this backen to test the remote.State implementation
func TestRemoteState(t *testing.T) {
	defer Reset()
	b := backend.TestBackendConfig(t, New(), hcl.EmptyBody())

	workspace := "workspace"

	// create a new workspace in this backend
	s, err := b.StateMgr(workspace)
	if err != nil {
		t.Fatal(err)
	}

	// force overwriting the remote state
	newState := statespkg.NewState()

	if err := s.WriteState(newState); err != nil {
		t.Fatal(err)
	}

	if err := s.PersistState(nil); err != nil {
		t.Fatal(err)
	}

	if err := s.RefreshState(); err != nil {
		t.Fatal(err)
	}
}
