// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/states"
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
		node := nodeCloseModule{Addr: addrs.Module{"child"}}
		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; !ok {
			t.Fatal("module.child should not be removed from state yet")
		}

		// the root module should do all the module cleanup
		node = nodeCloseModule{Addr: addrs.RootModule}
		diags = node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; ok {
			t.Fatal("module.child was not removed from state")
		}
	})
	t.Run("walkApplyNestedModules", func(t *testing.T) {
		state := states.NewState()
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey).Child("grandchild", addrs.NoKey))
		ctx := &MockEvalContext{
			StateState: state.SyncWrapper(),
		}
		node := nodeCloseModule{Addr: addrs.Module{"child"}}
		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; !ok {
			t.Fatal("module.child should not be removed from state yet")
		}
		if _, ok := state.Modules["module.child.module.grandchild"]; !ok {
			t.Fatal("module.child.module.grandchild should not be removed from state yet")
		}

		// the root module should do all the module cleanup
		node = nodeCloseModule{Addr: addrs.RootModule}
		diags = node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; ok {
			t.Fatal("module.child was not removed from state")
		}
		if _, ok := state.Modules["module.child.module.grandchild"]; ok {
			t.Fatal("module.child.module.grandchild was not removed from state")
		}
	})
	t.Run("walkApplyWithOutputs", func(t *testing.T) {
		state := states.NewState()
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey)).SetOutputValue("foo", cty.StringVal("bar"), false)
		ctx := &MockEvalContext{
			StateState: state.SyncWrapper(),
		}
		node := nodeCloseModule{Addr: addrs.Module{"child"}}
		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed even though
		// it has outputs.
		if _, ok := state.Modules["module.child"]; !ok {
			t.Fatal("module.child should not be removed from state yet")
		}

		// the root module should do all the module cleanup
		node = nodeCloseModule{Addr: addrs.RootModule}
		diags = node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; ok {
			t.Fatal("module.child was not removed from state")
		}
	})
	t.Run("walkApplyWithReferencedOutputs", func(t *testing.T) {
		state := states.NewState()
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey)).SetOutputValue("foo", cty.StringVal("bar"), false)
		ctx := &MockEvalContext{
			StateState: state.SyncWrapper(),
		}
		node := nodeCloseModule{Addr: addrs.Module{"child"}}
		diags := node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// module.child should not be removed, since we have referenced it from
		// the external references.
		if _, ok := state.Modules["module.child"]; !ok {
			t.Fatal("module.child should not be removed from state yet")
		}

		// the root module should do all the module cleanup
		node = nodeCloseModule{Addr: addrs.RootModule, ExternalReferences: []*addrs.Reference{
			{
				Subject: addrs.ModuleCallInstanceOutput{
					Call: addrs.ModuleCallInstance{
						Call: addrs.ModuleCall{
							Name: "child",
						},
						Key: addrs.NoKey,
					},
					Name: "foo",
				},
			},
		}}
		diags = node.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		// Since module.child has no resources, it should be removed
		if _, ok := state.Modules["module.child"]; !ok {
			t.Fatal("module.child should not have been removed from state at all")
		}
	})

	// walkImport is a no-op
	t.Run("walkImport", func(t *testing.T) {
		state := states.NewState()
		state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		ctx := &MockEvalContext{
			StateState: state.SyncWrapper(),
		}
		node := nodeCloseModule{Addr: addrs.Module{"child"}}

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
