// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package simple

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/pluggable"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

// TODO: Testing of locking with 2 clients once locking is fully implemented.

func TestFsStoreRemoteState(t *testing.T) {
	td := t.TempDir()
	t.Chdir(td)

	provider := Provider()

	plug, err := pluggable.NewPluggable(provider, fsStoreName)
	if err != nil {
		t.Fatal(err)
	}

	b := backend.TestBackendConfig(t, plug, hcl.EmptyBody())

	// The "default" state doesn't exist by default
	// (Note that this depends on the factory method used to get the provider above)
	stateIds, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		t.Fatal(wDiags.Err())
	}
	if len(stateIds) != 0 {
		t.Fatalf("unexpected response from Workspaces method: %#v", stateIds)
	}

	// create a new state using this backend
	newStateId := "foobar"
	emptyState := states.NewState()

	sMgr, sDiags := b.StateMgr(newStateId)
	if sDiags.HasErrors() {
		t.Fatal(sDiags.Err())
	}
	if err := sMgr.WriteState(emptyState); err != nil {
		t.Fatal(err)
	}
	if err := sMgr.PersistState(nil); err != nil {
		t.Fatal(err)
	}

	// force overwriting the remote state
	newState := states.NewState()
	newState.SetOutputValue(
		addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance),
		cty.StringVal("bar"),
		false)

	if err := sMgr.WriteState(newState); err != nil {
		t.Fatal(err)
	}

	if err := sMgr.PersistState(nil); err != nil {
		t.Fatal(err)
	}

	if err := sMgr.RefreshState(); err != nil {
		t.Fatal(err)
	}
}
