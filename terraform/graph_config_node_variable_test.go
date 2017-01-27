package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestGraphNodeConfigVariable_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigVariable)
	var _ dag.NamedVertex = new(GraphNodeConfigVariable)
	var _ graphNodeConfig = new(GraphNodeConfigVariable)
}

func TestGraphNodeConfigVariableFlat_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigVariableFlat)
	var _ dag.NamedVertex = new(GraphNodeConfigVariableFlat)
	var _ graphNodeConfig = new(GraphNodeConfigVariableFlat)
}
