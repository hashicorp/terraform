package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/depgraph2"
)

func TestGraphNodeConfigResource_impl(t *testing.T) {
	var _ depgraph.Node = new(GraphNodeConfigResource)
	var _ depgraph.NamedNode = new(GraphNodeConfigResource)
	var _ graphNodeConfig = new(GraphNodeConfigResource)
}
