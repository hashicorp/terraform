package terraform

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func TestNodeModuleVariablePath(t *testing.T) {
	n := &nodeModuleVariable{
		Addr: addrs.RootModuleInstance.InputVariable("foo"),
		Config: &configs.Variable{
			Name: "foo",
		},
	}

	want := addrs.RootModuleInstance
	got := n.Path()
	if got.String() != want.String() {
		t.Fatalf("wrong module address %s; want %s", got, want)
	}
}

func TestNodeModuleVariableReferenceableName(t *testing.T) {
	n := &nodeExpandModuleVariable{
		Addr: addrs.InputVariable{Name: "foo"},
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
		wantSelfPath := addrs.RootModuleInstance
		wantReferencePath := addrs.RootModuleInstance
		if got, want := gotSelfPath.String(), wantSelfPath.String(); got != want {
			t.Errorf("wrong self path\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := gotReferencePath.String(), wantReferencePath.String(); got != want {
			t.Errorf("wrong reference path\ngot:  %s\nwant: %s", got, want)
		}
	}

}

func TestNodeModuleVariableReference(t *testing.T) {
	n := &nodeExpandModuleVariable{
		Addr:   addrs.InputVariable{Name: "foo"},
		Module: addrs.RootModule.Child("bar"),
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

func TestNodeModuleVariableReference_grandchild(t *testing.T) {
	n := &nodeExpandModuleVariable{
		Addr:   addrs.InputVariable{Name: "foo"},
		Module: addrs.RootModule.Child("bar"),
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
