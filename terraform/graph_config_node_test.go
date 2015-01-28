package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

func TestGraphNodeConfigProvider_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigProvider)
	var _ dag.NamedVertex = new(GraphNodeConfigProvider)
	var _ graphNodeConfig = new(GraphNodeConfigProvider)
	var _ GraphNodeProvider = new(GraphNodeConfigProvider)
}

func TestGraphNodeConfigProvider_ProviderName(t *testing.T) {
	n := &GraphNodeConfigProvider{
		Provider: &config.ProviderConfig{Name: "foo"},
	}

	if v := n.ProviderName(); v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeConfigResource_impl(t *testing.T) {
	var _ dag.Vertex = new(GraphNodeConfigResource)
	var _ dag.NamedVertex = new(GraphNodeConfigResource)
	var _ graphNodeConfig = new(GraphNodeConfigResource)
	var _ GraphNodeProviderConsumer = new(GraphNodeConfigResource)
}

func TestGraphNodeConfigResource_ProvidedBy(t *testing.T) {
	n := &GraphNodeConfigResource{
		Resource: &config.Resource{Type: "aws_instance"},
	}

	if v := n.ProvidedBy(); v != "aws" {
		t.Fatalf("bad: %#v", v)
	}
}
