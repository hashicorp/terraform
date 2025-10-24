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

func TestBackendLocked(t *testing.T) {
	t.Setenv("TF_ACC", "1") // enable using the inmem state store

	// backend.TestBackendStateLocks assumes the default workspace exists
	// by default, so we need to make it exist using the method below.
	provider := ProviderWithDefaultWorkspace()

	plug1, err := pluggable.NewPluggable(provider, inMemStoreName)
	if err != nil {
		t.Fatal(err)
	}
	plug2, err := pluggable.NewPluggable(provider, inMemStoreName)
	if err != nil {
		t.Fatal(err)
	}

	b1 := backend.TestBackendConfig(t, plug1, nil)
	b2 := backend.TestBackendConfig(t, plug2, nil)

	backend.TestBackendStateLocks(t, b1, b2)
}

func TestRemoteState(t *testing.T) {
	t.Setenv("TF_ACC", "1") // enable using the inmem state store

	provider := Provider()

	plug, err := pluggable.NewPluggable(provider, inMemStoreName)
	if err != nil {
		t.Fatal(err)
	}

	b := backend.TestBackendConfig(t, plug, hcl.EmptyBody())

	// The default workspace doesn't exist by default
	// (Note that this depends on the factory method used to get the provider above)
	workspaces, wDiags := b.Workspaces()
	if wDiags.HasErrors() {
		t.Fatal(wDiags.Err())
	}
	if len(workspaces) != 0 {
		t.Fatalf("unexpected response from Workspaces method: %#v", workspaces)
	}

	// create a new workspace in this backend
	workspace := "workspace"
	emptyState := states.NewState()

	sMgr, sDiags := b.StateMgr(workspace)
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
