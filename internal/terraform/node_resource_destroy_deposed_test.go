package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestNodePlanDeposedResourceInstanceObject_Execute(t *testing.T) {
	deposedKey := states.NewDeposedKey()
	state := states.NewState()
	absResource := mustResourceInstanceAddr("test_instance.foo")
	state.Module(addrs.RootModuleInstance).SetResourceInstanceDeposed(
		absResource.Resource,
		deposedKey,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	p := testProvider("test")
	p.UpgradeResourceStateResponse = &providers.UpgradeResourceStateResponse{
		UpgradedState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("bar"),
		}),
	}
	ctx := &MockEvalContext{
		StateState:        state.SyncWrapper(),
		PrevRunStateState: state.DeepCopy().SyncWrapper(),
		RefreshStateState: state.DeepCopy().SyncWrapper(),
		ProviderProvider:  p,
		ProviderSchemaSchema: &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				"test_instance": {
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
					},
				},
			},
		},
		ChangesChanges: plans.NewChanges().SyncWrapper(),
	}

	node := NodePlanDeposedResourceInstanceObject{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			Addr: absResource,
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
			},
		},
		DeposedKey: deposedKey,
	}
	err := node.Execute(ctx, walkPlan)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !p.UpgradeResourceStateCalled {
		t.Errorf("UpgradeResourceState wasn't called; should've been called to upgrade the previous run's object")
	}
	if !p.ReadResourceCalled {
		t.Errorf("ReadResource wasn't called; should've been called to refresh the deposed object")
	}

	change := ctx.Changes().GetResourceInstanceChange(absResource, deposedKey)
	if got, want := change.ChangeSrc.Action, plans.Delete; got != want {
		t.Fatalf("wrong planned action\ngot:  %s\nwant: %s", got, want)
	}
}

func TestNodeDestroyDeposedResourceInstanceObject_Execute(t *testing.T) {
	deposedKey := states.NewDeposedKey()
	state := states.NewState()
	absResource := mustResourceInstanceAddr("test_instance.foo")
	state.Module(addrs.RootModuleInstance).SetResourceInstanceDeposed(
		absResource.Resource,
		deposedKey,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsJSON: []byte(`{"id":"bar"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	schema := &ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id": {
						Type:     cty.String,
						Computed: true,
					},
				},
			},
		},
	}

	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(schema)

	p.UpgradeResourceStateResponse = &providers.UpgradeResourceStateResponse{
		UpgradedState: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("bar"),
		}),
	}
	ctx := &MockEvalContext{
		StateState:           state.SyncWrapper(),
		ProviderProvider:     p,
		ProviderSchemaSchema: schema,
		ChangesChanges:       plans.NewChanges().SyncWrapper(),
	}

	node := NodeDestroyDeposedResourceInstanceObject{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			Addr: absResource,
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
			},
		},
		DeposedKey: deposedKey,
	}
	err := node.Execute(ctx, walkApply)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !state.Empty() {
		t.Fatalf("resources left in state after destroy")
	}
}

func TestNodeDestroyDeposedResourceInstanceObject_WriteResourceInstanceState(t *testing.T) {
	state := states.NewState()
	ctx := new(MockEvalContext)
	ctx.StateState = state.SyncWrapper()
	ctx.PathPath = addrs.RootModuleInstance
	mockProvider := mockProviderWithResourceTypeSchema("aws_instance", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Optional: true,
			},
		},
	})
	ctx.ProviderProvider = mockProvider
	ctx.ProviderSchemaSchema = mockProvider.ProviderSchema()

	obj := &states.ResourceInstanceObject{
		Value: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("i-abc123"),
		}),
		Status: states.ObjectReady,
	}
	node := &NodeDestroyDeposedResourceInstanceObject{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
			},
			Addr: mustResourceInstanceAddr("aws_instance.foo"),
		},
		DeposedKey: states.NewDeposedKey(),
	}
	err := node.writeResourceInstanceState(ctx, obj)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	checkStateString(t, state, `
aws_instance.foo: (1 deposed)
  ID = <not created>
  provider = provider["registry.terraform.io/hashicorp/aws"]
  Deposed ID 1 = i-abc123
	`)
}

func TestNodeDestroyDeposedResourceInstanceObject_ExecuteMissingState(t *testing.T) {
	p := simpleMockProvider()
	ctx := &MockEvalContext{
		StateState:           states.NewState().SyncWrapper(),
		ProviderProvider:     simpleMockProvider(),
		ProviderSchemaSchema: p.ProviderSchema(),
		ChangesChanges:       plans.NewChanges().SyncWrapper(),
	}

	node := NodeDestroyDeposedResourceInstanceObject{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			Addr: mustResourceInstanceAddr("test_object.foo"),
			NodeAbstractResource: NodeAbstractResource{
				ResolvedProvider: mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
			},
		},
		DeposedKey: states.NewDeposedKey(),
	}
	err := node.Execute(ctx, walkApply)

	if err == nil {
		t.Fatal("expected error")
	}
}
