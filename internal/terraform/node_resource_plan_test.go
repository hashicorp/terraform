package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNodePlannableResourceExecute(t *testing.T) {
	state := states.NewState()
	ctx := &MockEvalContext{
		StateState:               state.SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(),
	}

	t.Run("no config", func(t *testing.T) {
		node := NodePlannableResource{
			NodeAbstractResource: &NodeAbstractResource{
				Config: nil,
			},
			Addr: mustAbsResourceAddr("test_instance.foo"),
		}
		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}
		if !state.Empty() {
			t.Fatalf("expected no state, got:\n %s", state.String())
		}
	})

	t.Run("simple", func(t *testing.T) {

		node := NodePlannableResource{
			NodeAbstractResource: &NodeAbstractResource{
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
			Addr: mustAbsResourceAddr("test_instance.foo"),
		}
		diags := node.Execute(ctx, walkApply)
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
