package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
)

func TestNodeApplyableModuleVariablePath(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		PathValue: []string{"root", "child"},
		Config:    &config.Variable{Name: "foo"},
	}

	expected := []string{"root"}
	actual := n.Path()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v != %#v", actual, expected)
	}
}

func TestNodeApplyableModuleVariableReferenceableName(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		PathValue: []string{"root", "child"},
		Config:    &config.Variable{Name: "foo"},
	}

	expected := []string{"module.child.var.foo"}
	actual := n.ReferenceableName()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v != %#v", actual, expected)
	}
}

func TestNodeApplyableModuleVariableReference(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		PathValue: []string{"root", "child"},
		Config:    &config.Variable{Name: "foo"},
		Value: config.TestRawConfig(t, map[string]interface{}{
			"foo": `${var.foo}`,
		}),
	}

	expected := []string{"var.foo"}
	actual := n.References()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v != %#v", actual, expected)
	}
}

func TestNodeApplyableModuleVariableReference_grandchild(t *testing.T) {
	n := &NodeApplyableModuleVariable{
		PathValue: []string{"root", "child", "grandchild"},
		Config:    &config.Variable{Name: "foo"},
		Value: config.TestRawConfig(t, map[string]interface{}{
			"foo": `${var.foo}`,
		}),
	}

	expected := []string{"module.child.var.foo"}
	actual := n.References()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%#v != %#v", actual, expected)
	}
}
