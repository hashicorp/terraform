package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestModuleVariableTransformer(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	module := testModule(t, "transform-module-var-basic")

	{
		tf := &RootVariableTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &ModuleVariableTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformModuleVarBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestModuleVariableTransformer_nested(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	module := testModule(t, "transform-module-var-nested")

	{
		tf := &RootVariableTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &ModuleVariableTransformer{Config: module}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformModuleVarNestedStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformModuleVarBasicStr = `
module.child.var.value
`

const testTransformModuleVarNestedStr = `
module.child.module.child.var.value
module.child.var.value
`
