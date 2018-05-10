package terraform

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

func TestNodeApplyableModuleVariablePath(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey).InputVariable("foo"),
		Config: &configs.Variable{
			Name: "foo",
		},
	}

	expected := []string{"root"}
	actual := n.Path()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v != %#v", actual, expected)
	}
}

func TestNodeApplyableModuleVariableReferenceableName(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey).InputVariable("foo"),
		Config: &configs.Variable{
			Name: "foo",
		},
	}

	{
		expected := []addrs.Referenceable{
			addrs.InputVariable{Name: "foo"},
		}
		actual := n.ReferenceableAddrs()
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("%#v != %#v", actual, expected)
		}
	}

	{
		gotSelfPath, gotReferencePath := n.ReferenceOutside()
		wantSelfPath := addrs.RootModuleInstance.Child("child", addrs.NoKey)
		wantReferencePath := addrs.RootModuleInstance
		if got, want := gotSelfPath.String(), wantSelfPath.String(); got != want {
			t.Errorf("wrong self path\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := gotReferencePath.String(), wantReferencePath.String(); got != want {
			t.Errorf("wrong reference path\ngot:  %s\nwant: %s", got, want)
		}
	}

}

func TestNodeApplyableModuleVariableReference(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey).InputVariable("foo"),
		Config: &configs.Variable{
			Name: "foo",
		},
		Expr: &hclsyntax.ScopeTraversalExpr{
			Traversal: hcl.Traversal{
				hcl.TraverseRoot{Name: "var"},
				hcl.TraverseAttr{Name: "foo"},
			},
		},
	}

	want := []*addrs.Reference{
		{
			Subject: addrs.InputVariable{Name: "foo"},
		},
	}
	got := n.References()
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

func TestNodeApplyableModuleVariableReference_grandchild(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		Addr: addrs.RootModuleInstance.
			Child("child", addrs.NoKey).
			Child("grandchild", addrs.NoKey).
			InputVariable("foo"),
		Config: &configs.Variable{
			Name: "foo",
		},
		Expr: &hclsyntax.ScopeTraversalExpr{
			Traversal: hcl.Traversal{
				hcl.TraverseRoot{Name: "var"},
				hcl.TraverseAttr{Name: "foo"},
			},
		},
	}

	want := []*addrs.Reference{
		{
			Subject: addrs.InputVariable{Name: "foo"},
		},
	}
	got := n.References()
	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}
