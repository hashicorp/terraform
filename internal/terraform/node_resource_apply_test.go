// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodeExpandApplyableResourceExecute(t *testing.T) {
	state := states.NewState()
	t.Run("no config", func(t *testing.T) {
		ctx := &MockEvalContext{
			StateState:               state.SyncWrapper(),
			InstanceExpanderExpander: instances.NewExpander(nil),
		}

		node := &nodeExpandApplyableResource{
			NodeAbstractResource: &NodeAbstractResource{
				Addr:   mustConfigResourceAddr("test_instance.foo"),
				Config: nil,
			},
		}
		_, diags := node.DynamicExpand(ctx)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		state.PruneResourceHusks()
		if !state.Empty() {
			t.Fatalf("expected no state, got:\n %s", state.String())
		}
	})

	t.Run("simple", func(t *testing.T) {
		ctx := &MockEvalContext{
			StateState:               state.SyncWrapper(),
			InstanceExpanderExpander: instances.NewExpander(nil),
		}

		node := &nodeExpandApplyableResource{
			NodeAbstractResource: &NodeAbstractResource{
				Addr: mustConfigResourceAddr("test_instance.foo"),
				Config: &configs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_instance",
					Name: "foo",
				},
				ResolvedProvider: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
			},
		}
		_, diags := node.DynamicExpand(ctx)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		if state.Empty() {
			t.Fatal("expected resources in state, got empty state")
		}
		r := state.Resource(mustAbsResourceAddr("test_instance.foo"))
		if r == nil {
			t.Fatal("test_instance.foo not found in state")
		}
	})
}
