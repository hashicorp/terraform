package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeExpandModuleExecute(t *testing.T) {
	ctx := &MockEvalContext{
		InstanceExpanderExpander: instances.NewExpander(),
	}
	ctx.installSimpleEval()

	node := nodeExpandModule{
		Addr: addrs.Module{"child"},
		ModuleCall: &configs.ModuleCall{
			Count: hcltest.MockExprLiteral(cty.NumberIntVal(2)),
		},
	}

	err := node.Execute(ctx, walkApply)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !ctx.InstanceExpanderCalled {
		t.Fatal("did not expand")
	}
}

func TestNodeCloseModuleExecute(t *testing.T) {
	t.Run("walkApply", func(t *testing.T) {
		state := states.NewState()
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		ctx := &MockEvalContext{
			StateState: state.SyncWrapper(),
		}
		node := nodeCloseModule{addrs.Module{"child"}}
		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; ok {
			t.Fatal("module.child was not removed from state")
		}
	})

	// walkImport is a no-op
	t.Run("walkImport", func(t *testing.T) {
		state := states.NewState()
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		ctx := &MockEvalContext{
			StateState: state.SyncWrapper(),
		}
		node := nodeCloseModule{addrs.Module{"child"}}

		diags := node.Execute(ctx, walkImport)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}
		if _, ok := state.Modules["module.child"]; !ok {
			t.Fatal("module.child was removed from state, expected no-op")
		}
	})
}

func TestNodeValidateModuleExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := &MockEvalContext{
			InstanceExpanderExpander: instances.NewExpander(),
		}
		ctx.installSimpleEval()
		node := nodeValidateModule{
			nodeExpandModule{
				Addr: addrs.Module{"child"},
				ModuleCall: &configs.ModuleCall{
					Count: hcltest.MockExprLiteral(cty.NumberIntVal(2)),
				},
			},
		}

		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %v", diags.Err())
		}
	})

	t.Run("invalid count", func(t *testing.T) {
		ctx := &MockEvalContext{
			InstanceExpanderExpander: instances.NewExpander(),
		}
		ctx.installSimpleEval()
		node := nodeValidateModule{
			nodeExpandModule{
				Addr: addrs.Module{"child"},
				ModuleCall: &configs.ModuleCall{
					Count: hcltest.MockExprLiteral(cty.StringVal("invalid")),
				},
			},
		}

		err := node.Execute(ctx, walkApply)
		if err == nil {
			t.Fatal("expected error, got success")
		}
	})

}
