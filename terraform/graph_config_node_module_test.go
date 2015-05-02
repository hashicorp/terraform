package terraform

import (
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

	fg := g.(GraphNodeFlatGraph)

	actual := strings.TrimSpace(fg.FlattenGraph().String())
	expected := strings.TrimSpace(testGraphNodeModuleExpandFlattenStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testGraphNodeModuleExpandStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
  module inputs
module inputs
plan-destroy
`

const testGraphNodeModuleExpandFlattenStr = `
aws_instance.foo
plan-destroy
`
