package terraform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

func TestGraphNodeConfigModule_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigModule)
	var _ dag.NamedVertex = new(GraphNodeConfigModule)
	var _ graphNodeConfig = new(GraphNodeConfigModule)
	var _ GraphNodeExpandable = new(GraphNodeConfigModule)
}

func TestGraphNodeConfigModuleExpand(t *testing.T) {
	mod := testModule(t, "graph-node-module-expand")

	node := &GraphNodeConfigModule{
		Path:   []string{RootModuleName, "child"},
		Module: &config.Module{},
		Tree:   nil,
	}

	g, err := node.Expand(&BasicGraphBuilder{
		Steps: []GraphTransformer{
			&ConfigTransformer{Module: mod},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.Subgraph().String())
	expected := strings.TrimSpace(testGraphNodeModuleExpandStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphNodeConfigModuleExpandFlatten(t *testing.T) {
	mod := testModule(t, "graph-node-module-flatten")

	node := &GraphNodeConfigModule{
		Path:   []string{RootModuleName, "child"},
		Module: &config.Module{},
		Tree:   nil,
	}

	g, err := node.Expand(&BasicGraphBuilder{
		Steps: []GraphTransformer{
			&ConfigTransformer{Module: mod},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	fg := g.(GraphNodeFlattenable)

	actual := strings.TrimSpace(fg.FlattenGraph().String())
	expected := strings.TrimSpace(testGraphNodeModuleExpandFlattenStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphNodeModulFlatWrap_Name(t *testing.T) {
	n := &graphNodeModuleFlatWrap{
		graphNodeModuleWrappable: &testGraphNodeModuleWrappable{
			NameValue: "foo",
		},

		NamePrefix: "module.bar",
	}

	if v := n.Name(); v != "module.bar.foo" {
		t.Fatalf("bad: %s", v)
	}
}

func TestGraphNodeModulFlatWrap_DependentOn(t *testing.T) {
	n := &graphNodeModuleFlatWrap{
		graphNodeModuleWrappable: &testGraphNodeModuleWrappable{
			NameValue: "foo",
		},

		NamePrefix:        "module.bar",
		DependentOnPrefix: "module.bar",
	}

	actual := n.DependentOn()
	expected := []string{"module.bar.foo"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestGraphNodeModulFlatWrap_DependableName(t *testing.T) {
	n := &graphNodeModuleFlatWrap{
		graphNodeModuleWrappable: &testGraphNodeModuleWrappable{
			NameValue: "foo",
		},

		NamePrefix: "module.bar",
	}

	actual := n.DependableName()
	expected := []string{"module.bar.foo"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

type testGraphNodeModuleWrappable struct {
	NameValue string
}

func (n *testGraphNodeModuleWrappable) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeInvalid
}

func (n *testGraphNodeModuleWrappable) Name() string {
	return n.NameValue
}

func (n *testGraphNodeModuleWrappable) DependableName() []string {
	return []string{"foo"}
}

func (n *testGraphNodeModuleWrappable) DependentOn() []string {
	return []string{"foo"}
}

const testGraphNodeModuleExpandStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
  module inputs
module inputs
`

const testGraphNodeModuleExpandFlattenStr = `
module.child.aws_instance.foo
`
