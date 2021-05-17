package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeCountBoundaryExecute(t *testing.T) {

	// Create a state with a single instance (addrs.NoKey) of test_instance.foo
	state := states.NewState()
	state.Module(addrs.RootModuleInstance).SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "foo",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"type":"string","value":"hello"}`),
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)

	// Create a config that uses count to create 2 instances of test_instance.foo
	rc := &configs.Resource{
		Mode:  addrs.ManagedResourceMode,
		Type:  "test_instance",
		Name:  "foo",
		Count: hcltest.MockExprLiteral(cty.NumberIntVal(2)),
		Config: configs.SynthBody("", map[string]cty.Value{
			"test_string": cty.StringVal("hello"),
		}),
	}
	config := &configs.Config{
		Module: &configs.Module{
			ManagedResources: map[string]*configs.Resource{
				"test_instance.foo": rc,
			},
		},
	}

	ctx := &MockEvalContext{
		StateState: state.SyncWrapper(),
	}
	node := NodeCountBoundary{Config: config}

	diags := node.Execute(ctx, walkApply)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %s", diags.Err())
	}
	if !state.HasResources() {
		t.Fatal("resources missing from state")
	}

	// verify that the resource changed from test_instance.foo to
	// test_instance.foo.0 in the state
	actual := state.String()
	expected := "test_instance.foo.0:\n  ID = \n  provider = provider[\"registry.terraform.io/hashicorp/test\"]\n  type = string\n  value = hello"

	if actual != expected {
		t.Fatalf("wrong result: %s", actual)
	}
}
