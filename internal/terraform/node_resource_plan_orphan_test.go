// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodeResourcePlanOrphanExecute(t *testing.T) {
	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_object",
			Name: "foo",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			AttrsFlat: map[string]string{
				"test_string": "foo",
			},
			Status: states.ObjectReady,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)

	p := simpleMockProvider()
	p.ConfigureProvider(providers.ConfigureProviderRequest{})
	ctx := &MockEvalContext{
		StateState:               state.SyncWrapper(),
		RefreshStateState:        state.DeepCopy().SyncWrapper(),
		PrevRunStateState:        state.DeepCopy().SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(nil),
		ProviderProvider:         p,
		ProviderSchemaSchema: providers.ProviderSchema{
			ResourceTypes: map[string]providers.Schema{
				"test_object": {
					Block: simpleTestSchema(),
				},
			},
		},
		ChangesChanges: plans.NewChanges().SyncWrapper(),
		DeferralsState: deferring.NewDeferred(false),
	}

	node := NodePlannableResourceInstanceOrphan{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
			},
			Addr: mustResourceInstanceAddr("test_object.foo"),
		},
	}
	diags := node.Execute(ctx, walkApply)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
	if !state.Empty() {
		t.Fatalf("expected empty state, got %s", state.String())
	}
}

func TestNodeResourcePlanOrphanExecute_alreadyDeleted(t *testing.T) {
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_object",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetResourceInstanceCurrent(
		addr.Resource,
		&states.ResourceInstanceObjectSrc{
			AttrsFlat: map[string]string{
				"test_string": "foo",
			},
			Status: states.ObjectReady,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	refreshState := state.DeepCopy()
	prevRunState := state.DeepCopy()
	changes := plans.NewChanges()

	p := simpleMockProvider()
	p.ConfigureProvider(providers.ConfigureProviderRequest{})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.NullVal(p.GetProviderSchemaResponse.ResourceTypes["test_string"].Block.ImpliedType()),
	}
	ctx := &MockEvalContext{
		StateState:               state.SyncWrapper(),
		RefreshStateState:        refreshState.SyncWrapper(),
		PrevRunStateState:        prevRunState.SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(nil),
		ProviderProvider:         p,
		ProviderSchemaSchema: providers.ProviderSchema{
			ResourceTypes: map[string]providers.Schema{
				"test_object": {
					Block: simpleTestSchema(),
				},
			},
		},
		ChangesChanges: changes.SyncWrapper(),
		DeferralsState: deferring.NewDeferred(false),
	}

	node := NodePlannableResourceInstanceOrphan{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
			},
			Addr: mustResourceInstanceAddr("test_object.foo"),
		},
	}
	diags := node.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
	if !state.Empty() {
		t.Fatalf("expected empty state, got %s", state.String())
	}

	if got := prevRunState.ResourceInstance(addr); got == nil {
		t.Errorf("no entry for %s in the prev run state; should still be present", addr)
	}
	if got := refreshState.ResourceInstance(addr); got != nil {
		t.Errorf("refresh state has entry for %s; should've been removed", addr)
	}
	if got := changes.ResourceInstance(addr); got != nil {
		t.Errorf("there should be no change for the %s instance, got %s", addr, got.Action)
	}
}

// This test describes a situation which should not be possible, as this node
// should never work on deposed instances. However, a bug elsewhere resulted in
// this code path being exercised and triggered a panic. As a result, the
// assertions at the end of the test are minimal, as the behaviour (aside from
// not panicking) is unspecified.
func TestNodeResourcePlanOrphanExecute_deposed(t *testing.T) {
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_object",
		Name: "foo",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetResourceInstanceDeposed(
		addr.Resource,
		states.NewDeposedKey(),
		&states.ResourceInstanceObjectSrc{
			AttrsFlat: map[string]string{
				"test_string": "foo",
			},
			Status: states.ObjectReady,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)
	refreshState := state.DeepCopy()
	prevRunState := state.DeepCopy()
	changes := plans.NewChanges()

	p := simpleMockProvider()
	p.ConfigureProvider(providers.ConfigureProviderRequest{})
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.NullVal(p.GetProviderSchemaResponse.ResourceTypes["test_string"].Block.ImpliedType()),
	}
	ctx := &MockEvalContext{
		StateState:               state.SyncWrapper(),
		RefreshStateState:        refreshState.SyncWrapper(),
		PrevRunStateState:        prevRunState.SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(nil),
		ProviderProvider:         p,
		ProviderSchemaSchema: providers.ProviderSchema{
			ResourceTypes: map[string]providers.Schema{
				"test_object": {
					Block: simpleTestSchema(),
				},
			},
		},
		ChangesChanges: changes.SyncWrapper(),
		DeferralsState: deferring.NewDeferred(false),
	}

	node := NodePlannableResourceInstanceOrphan{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
			},
			Addr: mustResourceInstanceAddr("test_object.foo"),
		},
	}
	diags := node.Execute(ctx, walkPlan)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
}
