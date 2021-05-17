package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
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
	ctx := &MockEvalContext{
		StateState:               state.SyncWrapper(),
		RefreshStateState:        state.DeepCopy().SyncWrapper(),
		PrevRunStateState:        state.DeepCopy().SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(),
		ProviderProvider:         p,
		ProviderSchemaSchema: &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				"test_object": simpleTestSchema(),
			},
		},
		ChangesChanges: plans.NewChanges().SyncWrapper(),
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
	p.ReadResourceResponse = &providers.ReadResourceResponse{
		NewState: cty.NullVal(p.GetProviderSchemaResponse.ResourceTypes["test_string"].Block.ImpliedType()),
	}
	ctx := &MockEvalContext{
		StateState:               state.SyncWrapper(),
		RefreshStateState:        refreshState.SyncWrapper(),
		PrevRunStateState:        prevRunState.SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(),
		ProviderProvider:         p,
		ProviderSchemaSchema: &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				"test_object": simpleTestSchema(),
			},
		},
		ChangesChanges: changes.SyncWrapper(),
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
	if got := changes.ResourceInstance(addr); got == nil {
		t.Errorf("no entry for %s in the planned changes; should have a NoOp change", addr)
	} else {
		if got, want := got.Action, plans.NoOp; got != want {
			t.Errorf("planned change for %s has wrong action\ngot:  %s\nwant: %s", addr, got, want)
		}
	}

}
