package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestGraphNodeConfigResource_impl(t *testing.T) {
	var _ dag.Node = new(GraphNodeConfigResource)
	var _ dag.NamedNode = new(GraphNodeConfigResource)
	var _ graphNodeConfig = new(GraphNodeConfigResource)
}
