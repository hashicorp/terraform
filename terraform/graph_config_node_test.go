package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestGraphNodeConfigResource_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigResource)
	var _ dag.NamedVertex = new(GraphNodeConfigResource)
	var _ graphNodeConfig = new(GraphNodeConfigResource)
}
